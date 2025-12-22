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
        String b64 = Base64.getEncoder().encodeToString(text.getBytes());
        return rotate(b64,
                "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
                "NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm");
    }

    public static String dec(String encoded) {
        if (encoded == null || encoded.isEmpty()) return "";
        String rotated = rotate(encoded,
                "NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm",
                "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz");
        return new String(Base64.getDecoder().decode(rotated));
    }

    public static String getNiceDateTime() {
        return NICE_FORMAT.format(Instant.now());
    }

    public static long epochSeconds() {
        return Instant.now().getEpochSecond();
    }
}