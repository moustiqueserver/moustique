package moustique;

import java.time.LocalTime;
import java.time.format.DateTimeFormatter;
import java.util.concurrent.TimeUnit;

/**
 * Moustique Java Client - HTTP + MQTT Integration Test
 */
public class TestClient {

    private static final DateTimeFormatter TIME_FORMATTER = DateTimeFormatter.ofPattern("HH:mm:ss");

    private static boolean runTest(String protocol, String ip, String port, String username, String password, int mqttPort) {
        System.out.println("\n" + "=".repeat(60));
        System.out.println("Testing with " + protocol.toUpperCase() + " protocol");
        System.out.println("=".repeat(60) + "\n");

        boolean useMqtt = protocol.equals("mqtt");

        System.out.println("=== Moustique Java Client – Multi-tenant Test ===");
        System.out.println("Protocol: " + protocol.toUpperCase());
        System.out.println("Server: " + ip + ":" + port);
        if (useMqtt) {
            System.out.println("MQTT Port: " + mqttPort);
        }
        System.out.println("Username: " + username + "\n");

        // Create client with authentication
        MoustiqueClient client = new MoustiqueClient(
            ip,
            port,
            "JavaTestClient-" + protocol,
            username,
            password,
            useMqtt,
            mqttPort
        );

        System.out.println("Client ID: " + client.getClientName() + "\n");

        try {
            // 1. Publish
            System.out.println("1. Publishing message...");
            client.publish("/test/topic/java", "Hello from " + protocol.toUpperCase() + " test!").join();
            Thread.sleep(500);

            // 2. Set value
            System.out.println("2. Setting value...");
            client.putval("/test/value/java", "java-" + protocol + "-v1").join();
            Thread.sleep(500);

            // 3. Subscribe and receive
            System.out.println("3. Subscribing to /test/topic/java...");
            client.subscribe("/test/topic/java", msg -> {
                String timestamp = LocalTime.now().format(TIME_FORMATTER);
                System.out.println("[" + timestamp + "] MESSAGE → '" + msg.topic() + "': " + msg.message() + " (from " + msg.from() + ")");
            }).join();

            System.out.println("   Sending message to trigger callback...");
            Thread.sleep(1000); // Give subscription time to register
            client.publish("/test/topic/java", "This message should appear in callback! (" + protocol.toUpperCase() + ")").join();

            if (useMqtt) {
                System.out.println("   Waiting for MQTT messages (10 seconds)...");
                Thread.sleep(10000);
            } else {
                System.out.println("   Polling for HTTP messages (10 seconds)...");
                for (int i = 0; i < 20; i++) {
                    client.pickup().join();
                    Thread.sleep(500);
                }
            }

            System.out.println("\n=== " + protocol.toUpperCase() + " test complete! ===");

            if (useMqtt) {
                client.disconnect();
            }

            return true;

        } catch (Exception e) {
            System.err.println("\nError in " + protocol.toUpperCase() + " test: " + e.getMessage());
            e.printStackTrace();
            return false;
        }
    }

    public static void main(String[] args) {
        if (args.length < 4) {
            System.out.println("Usage: java moustique.TestClient <ip> <port> <username> <password> [mqtt_port]");
            System.out.println("Example: java moustique.TestClient localhost 33334 demo demo123");
            System.out.println("         java moustique.TestClient localhost 33334 demo demo123 1883");
            System.exit(1);
        }

        String ip = args[0];
        String port = args[1];
        String username = args[2];
        String password = args[3];
        int mqttPort = (args.length > 4) ? Integer.parseInt(args[4]) : 1883;

        System.out.println("\n" + "=".repeat(60));
        System.out.println("Moustique Java Client - HTTP + MQTT Integration Test");
        System.out.println("=".repeat(60));

        // Test HTTP
        boolean httpSuccess = runTest("http", ip, port, username, password, mqttPort);

        // Test MQTT
        boolean mqttSuccess = runTest("mqtt", ip, port, username, password, mqttPort);

        // Summary
        System.out.println("\n" + "=".repeat(60));
        System.out.println("TEST SUMMARY");
        System.out.println("=".repeat(60));
        System.out.println("HTTP test: " + (httpSuccess ? "✓ PASSED" : "✗ FAILED"));
        System.out.println("MQTT test: " + (mqttSuccess ? "✓ PASSED" : "✗ FAILED"));
        System.out.println("=".repeat(60) + "\n");

        System.exit((httpSuccess && mqttSuccess) ? 0 : 1);
    }
}
