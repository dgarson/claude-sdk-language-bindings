package com.dgarson.claude.sidecar;

import com.google.protobuf.ListValue;
import com.google.protobuf.NullValue;
import com.google.protobuf.Struct;
import com.google.protobuf.Value;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Utility methods for converting between Java maps/objects and protobuf {@link Struct}/{@link Value}.
 *
 * <p>Thread-safe: all methods are stateless.
 */
public final class ProtoUtil {

    private ProtoUtil() {}

    /**
     * Converts a protobuf {@link Struct} to a Java {@code Map<String, Object>}.
     */
    public static Map<String, Object> structToMap(Struct struct) {
        if (struct == null) {
            return Map.of();
        }
        Map<String, Object> result = new LinkedHashMap<>();
        for (Map.Entry<String, Value> entry : struct.getFieldsMap().entrySet()) {
            result.put(entry.getKey(), fromValue(entry.getValue()));
        }
        return result;
    }

    /**
     * Converts a Java {@code Map<String, Object>} to a protobuf {@link Struct}.
     */
    public static Struct mapToStruct(Map<String, Object> map) {
        if (map == null || map.isEmpty()) {
            return Struct.getDefaultInstance();
        }
        Struct.Builder builder = Struct.newBuilder();
        for (Map.Entry<String, Object> entry : map.entrySet()) {
            builder.putFields(entry.getKey(), toValue(entry.getValue()));
        }
        return builder.build();
    }

    /**
     * Converts a Java object to a protobuf {@link Value}.
     *
     * <p>Supports: null, Boolean, Number (as double), String, List, Map, Value (passthrough).
     */
    @SuppressWarnings("unchecked")
    public static Value toValue(Object obj) {
        if (obj == null) {
            return Value.newBuilder().setNullValue(NullValue.NULL_VALUE).build();
        }
        if (obj instanceof Value v) {
            return v;
        }
        if (obj instanceof Boolean b) {
            return Value.newBuilder().setBoolValue(b).build();
        }
        if (obj instanceof Number n) {
            return Value.newBuilder().setNumberValue(n.doubleValue()).build();
        }
        if (obj instanceof String s) {
            return Value.newBuilder().setStringValue(s).build();
        }
        if (obj instanceof List<?> list) {
            ListValue.Builder lb = ListValue.newBuilder();
            for (Object item : list) {
                lb.addValues(toValue(item));
            }
            return Value.newBuilder().setListValue(lb.build()).build();
        }
        if (obj instanceof Map<?, ?> map) {
            Struct struct = mapToStruct((Map<String, Object>) map);
            return Value.newBuilder().setStructValue(struct).build();
        }
        // Fallback: convert to string
        return Value.newBuilder().setStringValue(obj.toString()).build();
    }

    /**
     * Converts a protobuf {@link Value} to a Java object.
     */
    public static Object fromValue(Value value) {
        if (value == null) {
            return null;
        }
        return switch (value.getKindCase()) {
            case NULL_VALUE -> null;
            case BOOL_VALUE -> value.getBoolValue();
            case NUMBER_VALUE -> value.getNumberValue();
            case STRING_VALUE -> value.getStringValue();
            case STRUCT_VALUE -> structToMap(value.getStructValue());
            case LIST_VALUE -> {
                List<Object> list = new ArrayList<>();
                for (Value v : value.getListValue().getValuesList()) {
                    list.add(fromValue(v));
                }
                yield list;
            }
            case KIND_NOT_SET -> null;
        };
    }

    /**
     * Converts a protobuf {@link Value} to a {@code List<Object>}, returning an empty list if
     * the value is not a list.
     */
    public static List<Object> valueToList(Value value) {
        if (value == null || value.getKindCase() != Value.KindCase.LIST_VALUE) {
            return List.of();
        }
        List<Object> result = new ArrayList<>();
        for (Value v : value.getListValue().getValuesList()) {
            result.add(fromValue(v));
        }
        return result;
    }
}
