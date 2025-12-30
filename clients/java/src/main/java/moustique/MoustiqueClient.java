// clients/java/src/main/java/moustique/MoustiqueClient.java
package moustique;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.google.gson.JsonArray;
import com.google.gson.JsonElement;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.net.http.HttpRequest.BodyPublishers;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.function.Consumer;

// Optional MQTT support
import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttConnectOptions;
import org.eclipse.paho.client.mqttv3.MqttException;
import org.eclipse.paho.client.mqttv3.MqttMessage;
import org.eclipse.paho.client.mqttv3.IMqttDeliveryToken;
import org.eclipse.paho.client.mqttv3.MqttCallback;

public class MoustiqueClient {
    private final HttpClient httpClient;
    private final String baseUrl;
    private final String ip;
    private final String clientName;
    private final String username;
    private final String password;
    private final Map<String, List<Consumer<Message>>> callbacks = new HashMap<>();
    private final Gson gson = new Gson();

    // MQTT support
    private final boolean useMqtt;
    private final int mqttPort;
    private MqttClient mqttClient;
    private boolean mqttConnected = false;

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
        this(ip, port, clientName, null, null, false, 1883);
    }

    public MoustiqueClient(String ip, String port, String clientName, String username, String password) {
        this(ip, port, clientName, username, password, false, 1883);
    }

    public MoustiqueClient(String ip, String port, String clientName, String username, String password,
                          boolean useMqtt, int mqttPort) {
        this.httpClient = HttpClient.newHttpClient();
        this.baseUrl = "http://" + ip + ":" + port;
        this.ip = ip;
        this.clientName = clientName.isBlank()
                ? "java-" + System.nanoTime()
                : clientName.trim();
        this.username = username;
        this.password = password;
        this.useMqtt = useMqtt;
        this.mqttPort = mqttPort;

        System.out.println("Moustique Java client initialized: " + this.clientName);

        if (useMqtt) {
            initMqtt();
        }
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
                .thenAccept(res -> {
                    if (res.statusCode() >= 200 && res.statusCode() < 300) {
                        System.out.println("Published to " + topic);
                    } else {
                        System.err.println("⚠️  Publish FAILED to " + topic + " - HTTP " + res.statusCode());
                        System.err.println("    Response: " + res.body());
                    }
                });
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
                .thenAccept(res -> {
                    if (res.statusCode() >= 200 && res.statusCode() < 300) {
                        System.out.println("Putval: " + topic + " = " + value);
                    } else {
                        System.err.println("⚠️  Putval FAILED for " + topic + " - HTTP " + res.statusCode());
                        System.err.println("    Response: " + res.body());
                    }
                });
    }

    public CompletableFuture<Void> subscribe(String topic, Consumer<Message> callback) {
        callbacks.computeIfAbsent(topic, k -> new ArrayList<>()).add(callback);

        if (useMqtt && mqttConnected) {
            // MQTT subscription
            try {
                mqttClient.subscribe(topic);
                System.out.println("✓ Subscribed to " + topic + " via MQTT");
                return CompletableFuture.completedFuture(null);
            } catch (MqttException e) {
                System.err.println("MQTT subscribe failed, falling back to HTTP: " + e.getMessage());
                // Fall through to HTTP subscription
            }
        }

        // HTTP subscription
        Map<String, String> payload = addAuth(Map.of(
                "topic", Utils.enc(topic),
                "client", Utils.enc(clientName)
        ));

        return sendPost("/SUBSCRIBE", payload)
                .thenAccept(res -> System.out.println(clientName + " subscribed to " + topic));
    }

    public CompletableFuture<Void> pickup() {
        Map<String, String> payload = addAuth(Map.of("client", Utils.enc(clientName)));

        return sendPost("/PICKUP", payload)
                .thenAccept(response -> {
                    String body = response.body().trim();
                    if (body.isEmpty()) {
                        return;
                    }

                    String decrypted = Utils.dec(body);
                    if (decrypted.isEmpty()) {
                        return;
                    }

                    try {
                        // Parse JSON: {"topic": [{"topic":"...", "message":"...", "from":"..."}]}
                        JsonObject data = gson.fromJson(decrypted, JsonObject.class);

                        // Handle system message: server restart
                        if (data.has("/server/action/resubscribe")) {
                            System.out.println("⚠️  Server restarted - re-subscribing to all topics...");
                            resubscribe();
                        }

                        // Deliver regular messages to callbacks
                        for (Map.Entry<String, JsonElement> entry : data.entrySet()) {
                            String topic = entry.getKey();

                            // Skip system messages
                            if ("/server/action/resubscribe".equals(topic)) {
                                continue;
                            }

                            JsonArray messages = entry.getValue().getAsJsonArray();
                            List<Consumer<Message>> topicCallbacks = callbacks.get(topic);

                            if (topicCallbacks != null) {
                                for (JsonElement msgElement : messages) {
                                    JsonObject msgObj = msgElement.getAsJsonObject();
                                    String msgTopic = msgObj.has("topic") ? msgObj.get("topic").getAsString() : topic;
                                    String msgText = msgObj.has("message") ? msgObj.get("message").getAsString() : "";
                                    String msgFrom = msgObj.has("from") ? msgObj.get("from").getAsString() : "";

                                    Message msg = new Message(msgTopic, msgText, msgFrom);

                                    for (Consumer<Message> callback : topicCallbacks) {
                                        try {
                                            callback.accept(msg);
                                        } catch (Exception e) {
                                            System.err.println("Error in callback for topic '" + topic + "': " + e.getMessage());
                                        }
                                    }
                                }
                            }
                        }
                    } catch (Exception e) {
                        System.err.println("JSON parse error: " + e.getMessage());
                        System.err.println("Decrypted text: " + decrypted);
                    }
                })
                .exceptionally(ex -> {
                    // Suppress errors during normal operation (empty responses are common)
                    if (ex.getCause() != null && ex.getCause().getMessage() != null &&
                        !ex.getCause().getMessage().contains("Illegal base64")) {
                        System.err.println("Pickup error: " + ex.getMessage());
                    }
                    return null;
                });
    }

    private void resubscribe() {
        for (String topic : new ArrayList<>(callbacks.keySet())) {
            if (useMqtt && mqttConnected) {
                // MQTT resubscribe
                try {
                    mqttClient.subscribe(topic);
                    System.out.println("✓ Re-subscribed to " + topic + " via MQTT");
                } catch (MqttException e) {
                    System.err.println("Failed to resubscribe to '" + topic + "' via MQTT: " + e.getMessage());
                }
            } else {
                // HTTP resubscribe
                Map<String, String> payload = addAuth(Map.of(
                        "topic", Utils.enc(topic),
                        "client", Utils.enc(clientName)
                ));

                sendPost("/SUBSCRIBE", payload)
                        .thenAccept(res -> System.out.println("✓ Re-subscribed to " + topic))
                        .exceptionally(ex -> {
                            System.err.println("Failed to resubscribe to '" + topic + "': " + ex.getMessage());
                            return null;
                        });
            }
        }
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

    // MQTT Support Methods

    private void initMqtt() {
        try {
            String broker = "tcp://" + ip + ":" + mqttPort;
            mqttClient = new MqttClient(broker, clientName);

            MqttConnectOptions options = new MqttConnectOptions();
            options.setCleanSession(true);

            if (username != null && password != null) {
                options.setUserName(username);
                options.setPassword(password.toCharArray());
            }

            mqttClient.setCallback(new MqttCallback() {
                @Override
                public void connectionLost(Throwable cause) {
                    mqttConnected = false;
                    System.err.println("⚠️  MQTT connection lost: " + cause.getMessage());
                }

                @Override
                public void messageArrived(String topic, MqttMessage message) {
                    try {
                        String payloadStr = new String(message.getPayload());
                        String msgTopic, msgText, msgFrom;

                        // Try to parse as JSON first (for compatibility)
                        // If it fails, treat as plaintext (standard MQTT)
                        try {
                            JsonObject msgObj = gson.fromJson(payloadStr, JsonObject.class);
                            msgTopic = msgObj.has("topic") ? msgObj.get("topic").getAsString() : topic;
                            msgText = msgObj.has("message") ? msgObj.get("message").getAsString() : "";
                            msgFrom = msgObj.has("from") ? msgObj.get("from").getAsString() : "";
                        } catch (Exception e) {
                            // Standard MQTT: plaintext payload
                            msgTopic = topic;
                            msgText = payloadStr;
                            msgFrom = "";
                        }

                        Message msg = new Message(msgTopic, msgText, msgFrom);

                        // Find matching callbacks
                        for (Map.Entry<String, List<Consumer<Message>>> entry : callbacks.entrySet()) {
                            if (topicMatches(entry.getKey(), msgTopic)) {
                                for (Consumer<Message> callback : entry.getValue()) {
                                    try {
                                        callback.accept(msg);
                                    } catch (Exception e) {
                                        System.err.println("Error in callback for topic '" + topic + "': " + e.getMessage());
                                    }
                                }
                            }
                        }
                    } catch (Exception e) {
                        System.err.println("Error processing MQTT message: " + e.getMessage());
                    }
                }

                @Override
                public void deliveryComplete(IMqttDeliveryToken token) {
                    // Not needed for subscribers
                }
            });

            mqttClient.connect(options);
            mqttConnected = true;
            System.out.println("✓ Connected to MQTT broker at " + broker);

        } catch (MqttException e) {
            System.err.println("MQTT connection failed: " + e.getMessage());
            System.err.println("Falling back to HTTP mode");
        }
    }

    private boolean topicMatches(String pattern, String topic) {
        // Simple MQTT wildcard matching (+ for single level, # for multi-level)
        String[] patternParts = pattern.split("/");
        String[] topicParts = topic.split("/");

        if (patternParts.length > topicParts.length && !patternParts[patternParts.length - 1].equals("#")) {
            return false;
        }

        for (int i = 0; i < patternParts.length; i++) {
            if (patternParts[i].equals("#")) {
                return true; // Match everything after
            }
            if (i >= topicParts.length) {
                return false;
            }
            if (patternParts[i].equals("+")) {
                continue; // Match single level
            }
            if (!patternParts[i].equals(topicParts[i])) {
                return false;
            }
        }

        return patternParts.length == topicParts.length ||
               (patternParts.length > 0 && patternParts[patternParts.length - 1].equals("#"));
    }

    public void disconnect() {
        if (mqttClient != null && mqttConnected) {
            try {
                mqttClient.disconnect();
                mqttClient.close();
                mqttConnected = false;
                System.out.println("MQTT client disconnected");
            } catch (MqttException e) {
                System.err.println("Error disconnecting MQTT client: " + e.getMessage());
            }
        }
    }
}