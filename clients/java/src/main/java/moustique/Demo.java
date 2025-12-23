package moustique;

import java.util.concurrent.TimeUnit;

public class Demo {
    public static void main(String[] args) throws Exception {
        MoustiqueClient client = new MoustiqueClient("127.0.0.1", "33335", "JavaDemo");

        client.subscribe("/test/topic", msg -> {
            System.out.println("[JAVA] " + msg.topic() + ": " + msg.message() + " (from " + msg.from() + ")");
        });

        client.publish("/test/topic", "Hej från Java-klienten!").join();
        client.putval("/test/value", "java-value-123").join();

        // Poll every second
        while (true) {
            client.pickup().join();
            TimeUnit.SECONDS.sleep(1);
        }
    }
}