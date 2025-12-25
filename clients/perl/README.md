# Moustique Perl Client

Perl client library for the Moustique message broker.

## Installation

Copy `Moustique.pm` to your Perl library path or include it in your project:

```perl
use lib '/path/to/moustique/clients/perl';
use Moustique;
```

## Usage

### Basic Example

```perl
use Moustique;

# Create client
my $client = Moustique->new(
    ip => "localhost",
    port => "33334",
    name => "my-app"
);

# Subscribe to a topic
$client->subscribe("/sensors/temperature", sub {
    my ($topic, $message, $from) = @_;
    print "Temperature: $message from $from\n";
});

# Publish a message
$client->publish("/sensors/temperature", "23.5", "sensor-01");

# Poll for messages (run in loop)
while (1) {
    $client->tick();
    sleep(1);
}
```

### With Authentication (Multi-tenant)

```perl
use Moustique;

# Method 1: Pass credentials to constructor
my $client = Moustique->new(
    ip => "localhost",
    port => "33334",
    name => "my-app",
    username => "alice",
    password => "secret123"
);

# Method 2: Set global credentials
$Moustique::GLOBAL_USERNAME = "alice";
$Moustique::GLOBAL_PASSWORD = "secret123";

my $client = Moustique->new(
    ip => "localhost",
    port => "33334",
    name => "my-app"
);

# Now all operations use authentication
$client->publish("/private/data", "sensitive info", "my-app");
```

### Class Methods (No Object)

```perl
use Moustique;

# Set global credentials for class methods
$Moustique::GLOBAL_USERNAME = "alice";
$Moustique::GLOBAL_PASSWORD = "secret123";

# Publish without creating object
Moustique::publish_nothread("localhost", "33334", "/topic", "message", "sender");

# Put value
Moustique::publish_nothread_put("localhost", "33334", "/config/key", "value", "setter");

# Get value
my $value = Moustique::getval("localhost", "33334", "/config/key");
print "Value: " . $value->{message} . "\n";

# Get server stats
my $stats = Moustique::getstats("localhost", "33334");

# Get server version
my $version = Moustique::getversion("localhost", "33334");
```

### Instance Methods

```perl
my $client = Moustique->new(
    ip => "localhost",
    port => "33334",
    name => "my-app",
    username => "alice",
    password => "secret123"
);

# Publish
$client->publish("/topic", "message", "sender");

# Subscribe
$client->subscribe("/topic", sub {
    my ($topic, $message, $from) = @_;
    print "$topic: $message from $from\n";
});

# Get value
my $value = $client->get_val("/config/setting");

# Resubscribe (useful after server restart)
$client->resubscribe();

# Pickup messages manually
$client->pickup();

# Or use tick (alias for pickup)
$client->tick();

# Get client name
my $name = $client->get_client_name();
```

### Wildcard Subscriptions

```perl
# Subscribe to all sensors in any room
$client->subscribe("/home/+/temperature", \&on_temperature);

# Subscribe to everything under /sensors/
$client->subscribe("/sensors/#", \&on_sensor_data);

sub on_temperature {
    my ($topic, $message, $from) = @_;
    print "Temperature: $message\n";
}

sub on_sensor_data {
    my ($topic, $message, $from) = @_;
    print "Sensor data: $topic = $message\n";
}
```

## API Reference

### Constructor

```perl
my $client = Moustique->new(%params);
```

Parameters:
- `ip` - Server IP address (required)
- `port` - Server port (required)
- `name` - Client name (optional, auto-generated if not provided)
- `username` - Username for authentication (optional)
- `password` - Password for authentication (optional)

### Instance Methods

- `publish($topic, $message, $from)` - Publish a message
- `subscribe($topic, $callback)` - Subscribe to a topic
- `get_val($valname)` - Get stored value
- `pickup()` - Poll for new messages
- `tick()` - Alias for pickup()
- `resubscribe()` - Resubscribe to all topics
- `get_client_name()` - Get client name

### Class Methods

All class methods accept optional `$username` and `$password` as the last two parameters. If not provided, they fall back to global credentials.

- `publish_nothread($ip, $port, $topic, $message, $from, $username, $password)` - Publish without object
- `publish_nothread_put($ip, $port, $topic, $value, $from, $username, $password)` - Put value without object
- `getval($ip, $port, $valname, $username, $password)` - Get value
- `get_vals_by_regex($ip, $port, $regex, $username, $password)` - Search values by pattern
- `getversion($ip, $port, $password)` - Get server version
- `getstats($ip, $port, $password)` - Get server statistics
- `getclients($ip, $port, $password)` - Get active clients
- `gettopics($ip, $port, $password)` - Get all topics

### Global Variables

```perl
# Set default credentials for all class methods
$Moustique::GLOBAL_USERNAME = "alice";
$Moustique::GLOBAL_PASSWORD = "secret123";
```

## Examples

### IoT Sensor

```perl
use Moustique;

my $client = Moustique->new(
    ip => "192.168.1.100",
    port => "33334",
    name => "temperature-sensor"
);

while (1) {
    my $temp = read_temperature(); # Your sensor reading function
    $client->publish("/sensors/living_room/temperature", $temp, "sensor-01");
    sleep(60); # Every minute
}
```

### Log Aggregator

```perl
use Moustique;

my $client = Moustique->new(
    ip => "logs.example.com",
    port => "33334",
    name => "app-logger",
    username => "logger",
    password => "secret"
);

# Subscribe to all error logs
$client->subscribe("/logs/+/error", sub {
    my ($topic, $message, $from) = @_;
    log_to_file("error.log", "$topic: $message from $from");
});

# Poll for messages
while (1) {
    $client->pickup();
    sleep(1);
}
```

### Configuration Sync

```perl
use Moustique;

# Set global credentials
$Moustique::GLOBAL_USERNAME = "config-service";
$Moustique::GLOBAL_PASSWORD = "config-pass";

# Store config
Moustique::publish_nothread_put(
    "localhost", "33334",
    "/config/database/host",
    "db.example.com",
    "config-updater"
);

# Read config
my $db_host = Moustique::getval("localhost", "33334", "/config/database/host");
print "DB Host: " . $db_host->{message} . "\n";
```

## Encoding

The Perl client uses ROT13 + Base64 encoding (ROT13 first, then Base64) to match the server's encoding scheme. This is handled automatically by the library.

## Troubleshooting

### Authentication Errors

If you get 401 errors:
1. Check that credentials are set correctly
2. Verify the user exists on the server
3. Ensure `allow_public` is `false` if authentication is required

```perl
# Debug: Print what's being sent
use Data::Dumper;
print Dumper($client);
```

### Connection Issues

```perl
# Test basic connectivity
use Moustique;
my $version = Moustique::getversion("localhost", "33334");
print "Server version: $version\n";
```

## Notes

- The client automatically generates a unique client name if not provided
- Callbacks are stored per topic and can have multiple handlers
- The `pickup()` method should be called regularly (e.g., every second) to process incoming messages
- Global credentials are used as fallback when credentials are not explicitly provided

## License

Same as Moustique server - MIT License
