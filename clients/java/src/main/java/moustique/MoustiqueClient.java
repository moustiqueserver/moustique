// clients/java/src/main/java/moustique/MoustiqueClient.java
package moustique;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.net.http.HttpRequest.BodyPublishers;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.function.Consumer;

public class MoustiqueClient {
    private final HttpClient httpClient;
    private final String baseUrl;
    private final String clientName;
    private final String username;
    private final String password;
    private final Map<String, List<Consumer<Message>>> callbacks = new HashMap<>();

    public static class Message {
        public final String topic;
        public final String message;
        public final String from;

        public Message(String topic, String message, String from) {
            this.topic = topic;
            this.message = message;
            this.from = from;
        }

         public String topic() {
            return topic;
        }

        public String message() {
            return message;
        }

        public String from() {
            return from;
        }

        @Override
        public String toString() {
            return "Message[topic=" + topic + ", message=" + message + ", from=" + from + "]";
        }
    }

    public MoustiqueClient(String ip, String port, String clientName) {
        this(ip, port, clientName, null, null);
    }

    public MoustiqueClient(String ip, String port, String clientName, String username, String password) {
        this.httpClient = HttpClient.newHttpClient();
        this.baseUrl = "http://" + ip + ":" + port;
        this.clientName = clientName.isBlank()
                ? "java-" + System.nanoTime()
                : clientName.trim();
        this.username = username;
        this.password = password;
        System.out.println("Moustique Java client initialized: " + this.clientName);
    }

    private Map<String, String> addAuth(Map<String, String> payload) {
        if (username != null && password != null && !username.isBlank() && !password.isBlank()) {
            Map<String, String> withAuth = new HashMap<>(payload);
            withAuth.put("username", Utils.enc(username));
            withAuth.put("password", Utils.enc(password));
            return withAuth;
        }
        return payload;
    }

    public CompletableFuture<Void> publish(String topic, String message) {
        Map<String, String> payload = addAuth(Map.of(
                "topic", Utils.enc(topic),
                "message", Utils.enc(message),
                "updated_time", Utils.enc(String.valueOf(Utils.epochSeconds())),
                "updated_nicedatetime", Utils.enc(Utils.getNiceDateTime()),
                "from", Utils.enc(clientName)
        ));

        return sendPost("/POST", payload)
                .thenAccept(res -> System.out.println("Published to " + topic));
    }

    public CompletableFuture<Void> putval(String topic, String value) {
        Map<String, String> payload = addAuth(Map.of(
                "valname", Utils.enc(topic),
                "val", Utils.enc(value),
                "updated_time", Utils.enc(String.valueOf(Utils.epochSeconds())),
                "updated_nicedatetime", Utils.enc(Utils.getNiceDateTime()),
                "from", Utils.enc(clientName)
        ));

        return sendRequest("PUT", "/PUTVAL", payload)
                .thenAccept(res -> System.out.println("Putval: " + topic + " = " + value));
    }

    public CompletableFuture<Void> subscribe(String topic, Consumer<Message> callback) {
        Map<String, String> payload = addAuth(Map.of(
                "topic", Utils.enc(topic),
                "client", Utils.enc(clientName)
        ));

        callbacks.computeIfAbsent(topic, k -> new ArrayList<>()).add(callback);

        return sendPost("/SUBSCRIBE", payload)
                .thenAccept(res -> System.out.println(clientName + " subscribed to " + topic));
    }

    public CompletableFuture<Void> pickup() {
        Map<String, String> payload = addAuth(Map.of("client", Utils.enc(clientName)));

        return sendPost("/PICKUP", payload)
                .thenAccept(response -> {
                    String decrypted = Utils.dec(response.body().trim());
                    if (decrypted.isEmpty()) {
                        return;
                    }

                    System.out.println("Raw pickup data: " + decrypted);

                    // TODO: Replace with real JSON library (Jackson/Gson) in production
                    // For now, just log the raw data – parsing can be added later
                    // Example expected format: {"topic": [{"topic":"...", "message":"...", "from":"..."}]}
                })
                .exceptionally(ex -> {
                    System.err.println("Pickup error: " + ex.getMessage());
                    return null;
                });
    }

    private CompletableFuture<HttpResponse<String>> sendPost(String endpoint, Map<String, String> formData) {
        return sendRequest("POST", endpoint, formData);
    }

    private CompletableFuture<HttpResponse<String>> sendRequest(String method, String endpoint, Map<String, String> formData) {
        var builder = HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + endpoint))
                .header("Content-Type", "application/x-www-form-urlencoded");

        if (formData != null && !formData.isEmpty()) {
            String body = formData.entrySet().stream()
                    .map(e -> e.getKey() + "=" + e.getValue())
                    .reduce((a, b) -> a + "&" + b)
                    .orElse("");
            builder.method(method, BodyPublishers.ofString(body));
        } else {
            builder.method(method, BodyPublishers.noBody());
        }

        return httpClient.sendAsync(builder.build(), HttpResponse.BodyHandlers.ofString());
    }

    public String getClientName() {
        return clientName;
    }
}