package com.dgarson.claude.sidecar;

import claude_sidecar.v1.Sidecar.ServerEvent;
import claude_sidecar.v1.Sidecar.TurnBoundary;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.locks.Condition;
import java.util.concurrent.locks.ReentrantLock;

/**
 * Thread-safe event multiplexer that routes {@link ServerEvent}s to global and per-request
 * subscribers. Mirrors Go's event_mux.go.
 *
 * <p>Global subscribers receive all events. Per-request subscribers receive only events matching
 * their request ID. Per-request subscriptions are automatically closed on TURN_END.</p>
 */
final class EventMux {

    private final EventQueue inbound = new EventQueue();
    private final Object lock = new Object();
    private final Map<String, List<Subscription>> byRequest = new HashMap<>();
    private final List<Subscription> global = new ArrayList<>();
    private boolean closed;
    private final Thread dispatchThread;

    EventMux() {
        dispatchThread = Thread.ofVirtual().name("event-mux-dispatch").start(this::run);
    }

    /** Enqueue an event for dispatch to subscribers. */
    void enqueue(ServerEvent event) {
        inbound.push(event);
    }

    /** Stop accepting events and close all subscriptions. */
    void close() {
        inbound.close();
    }

    /** Subscribe to all events. The returned subscription's queue will receive every event. */
    Subscription subscribeAll(int bufferSize) {
        Subscription sub = new Subscription(bufferSize);
        synchronized (lock) {
            if (closed) {
                sub.close();
                return sub;
            }
            global.add(sub);
        }
        return sub;
    }

    /** Subscribe to events for a specific request ID. */
    Subscription subscribeRequest(String requestId, int bufferSize) {
        Subscription sub = new Subscription(bufferSize);
        synchronized (lock) {
            if (closed) {
                sub.close();
                return sub;
            }
            byRequest.computeIfAbsent(requestId, k -> new ArrayList<>()).add(sub);
        }
        return sub;
    }

    /** Unsubscribe a per-request subscription and close it. */
    void unsubscribeRequest(String requestId, Subscription sub) {
        if (sub == null || requestId == null || requestId.isEmpty()) {
            return;
        }
        synchronized (lock) {
            List<Subscription> subs = byRequest.get(requestId);
            if (subs != null) {
                subs.remove(sub);
                if (subs.isEmpty()) {
                    byRequest.remove(requestId);
                }
            }
        }
        sub.close();
    }

    /** Main dispatch loop, runs on a dedicated thread. */
    private void run() {
        while (true) {
            ServerEvent event = inbound.pop();
            if (event == null) {
                closeAll();
                return;
            }
            List<Subscription> targets = subscriptionsFor(event);
            for (Subscription sub : targets) {
                sub.enqueue(event);
            }
            if (isTurnEnd(event)) {
                closeRequest(event.getRequestId());
            }
        }
    }

    private List<Subscription> subscriptionsFor(ServerEvent event) {
        synchronized (lock) {
            List<Subscription> subs = new ArrayList<>(global);
            String requestId = event.getRequestId();
            if (requestId != null && !requestId.isEmpty()) {
                List<Subscription> requestSubs = byRequest.get(requestId);
                if (requestSubs != null) {
                    subs.addAll(requestSubs);
                }
            }
            return subs;
        }
    }

    private void closeRequest(String requestId) {
        if (requestId == null || requestId.isEmpty()) {
            return;
        }
        List<Subscription> subs;
        synchronized (lock) {
            subs = byRequest.remove(requestId);
        }
        if (subs != null) {
            for (Subscription sub : subs) {
                sub.close();
            }
        }
    }

    private void closeAll() {
        List<Subscription> allSubs;
        synchronized (lock) {
            if (closed) {
                return;
            }
            closed = true;
            allSubs = new ArrayList<>(global);
            for (List<Subscription> list : byRequest.values()) {
                allSubs.addAll(list);
            }
            global.clear();
            byRequest.clear();
        }
        for (Subscription sub : allSubs) {
            sub.close();
        }
    }

    private static boolean isTurnEnd(ServerEvent event) {
        if (!event.hasTurn()) {
            return false;
        }
        return event.getTurn().getKind() == TurnBoundary.Kind.TURN_END;
    }

    // -------------------------------------------------------------------------
    // Inner: thread-safe blocking queue with close semantics
    // -------------------------------------------------------------------------

    static final class EventQueue {
        private final ReentrantLock queueLock = new ReentrantLock();
        private final Condition notEmpty = queueLock.newCondition();
        private final List<ServerEvent> items = new ArrayList<>();
        private boolean queueClosed;

        /** Push an event. Returns false if the queue is closed. */
        boolean push(ServerEvent event) {
            queueLock.lock();
            try {
                if (queueClosed) {
                    return false;
                }
                items.add(event);
                notEmpty.signal();
                return true;
            } finally {
                queueLock.unlock();
            }
        }

        /** Blocking pop. Returns null when the queue is closed and drained. */
        ServerEvent pop() {
            queueLock.lock();
            try {
                while (items.isEmpty() && !queueClosed) {
                    notEmpty.await();
                }
                if (items.isEmpty()) {
                    return null;
                }
                return items.remove(0);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                return null;
            } finally {
                queueLock.unlock();
            }
        }

        void close() {
            queueLock.lock();
            try {
                queueClosed = true;
                notEmpty.signalAll();
            } finally {
                queueLock.unlock();
            }
        }
    }

    // -------------------------------------------------------------------------
    // Inner: a subscription backed by a blocking queue, bridged to a
    // LinkedBlockingQueue for consumer convenience.
    // -------------------------------------------------------------------------

    static final class Subscription {
        private final EventQueue queue = new EventQueue();
        private final BlockingQueue<ServerEvent> out;
        private volatile boolean subClosed;

        Subscription(int bufferSize) {
            this.out = new LinkedBlockingQueue<>(Math.max(bufferSize, 1));
            Thread.ofVirtual().name("subscription-pump").start(this::pump);
        }

        private void pump() {
            while (true) {
                ServerEvent event = queue.pop();
                if (event == null) {
                    return;
                }
                try {
                    out.put(event);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    return;
                }
            }
        }

        void enqueue(ServerEvent event) {
            queue.push(event);
        }

        /**
         * Returns the output queue. Consumers should poll/take from this queue.
         * The queue will stop producing when the subscription is closed.
         */
        BlockingQueue<ServerEvent> queue() {
            return out;
        }

        void close() {
            if (!subClosed) {
                subClosed = true;
                queue.close();
            }
        }

        boolean isClosed() {
            return subClosed;
        }
    }
}
