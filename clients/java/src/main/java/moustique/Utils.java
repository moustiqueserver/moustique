// clients/java/src/main/java/moustique/Utils.java
package moustique;

import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.Base64;

public class Utils {
    private static final DateTimeFormatter NICE_FORMAT = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")
            .withZone(ZoneId.systemDefault());

    private static String rotate(String s, String from, String to) {
        StringBuilder sb = new StringBuilder();
        for (char c : s.toCharArray()) {
            int idx = from.indexOf(c);
            sb.append(idx != -1 ? to.charAt(idx) : c);
        }
        return sb.toString();
    }

    public static String enc(String text) {
        if (text == null || text.isEmpty()) return "";
        // Must match server encoding: ROT13 first, then Base64
        String rot13 = rotate(text,
                "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
                "NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm");
        return Base64.getEncoder().encodeToString(rot13.getBytes());
    }

    public static String dec(String encoded) {
        if (encoded == null || encoded.isEmpty()) return "";
        try {
            // Reverse of encode: Base64 decode first, then ROT13
            byte[] decoded = Base64.getDecoder().decode(encoded);
            String b64Text = new String(decoded);
            return rotate(b64Text,
                    "NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm",
                    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz");
        } catch (IllegalArgumentException e) {
            // Invalid base64 - return empty string
            return "";
        }
    }

    public static String getNiceDateTime() {
        return NICE_FORMAT.format(Instant.now());
    }

    public static long epochSeconds() {
        return Instant.now().getEpochSecond();
    }
}