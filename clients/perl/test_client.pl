#!/usr/bin/env perl
# clients/perl/test_client.pl

use strict;
use warnings;
use FindBin;
use lib "$FindBin::Bin";
use Moustique;
use Time::HiRes qw(sleep);

my $server_ip = $ARGV[0] || "localhost";
my $server_port = $ARGV[1] || "33334";
my $username = $ARGV[2] || "testuser";
my $password = $ARGV[3] || "testpass";
my $mqtt_port = $ARGV[4] || "1883";

print "=" x 60 . "\n";
print "Moustique Perl Client Test\n";
print "=" x 60 . "\n\n";

# Test both HTTP and MQTT
my $http_success = run_test("http", $server_ip, $server_port, $username, $password, $mqtt_port);
my $mqtt_success = run_test("mqtt", $server_ip, $server_port, $username, $password, $mqtt_port);

print "\n" . "=" x 60 . "\n";
print "Test Summary:\n";
print "=" x 60 . "\n";
print "HTTP: " . ($http_success ? "✓ PASS" : "✗ FAIL") . "\n";
print "MQTT: " . ($mqtt_success ? "✓ PASS" : "✗ FAIL") . "\n";
print "=" x 60 . "\n";

exit(($http_success && $mqtt_success) ? 0 : 1);

sub run_test {
    my ($protocol, $ip, $port, $user, $pass, $mqtt_port) = @_;
    my $use_mqtt = ($protocol eq "mqtt");

    print "\n--- Testing Perl client with $protocol ---\n";

    # Create client
    my $client;
    if ($use_mqtt) {
        $client = Moustique->new(
            ip => $ip,
            port => $port,
            name => "PerlTest-$protocol",
            username => $user,
            password => $pass,
            use_mqtt => 1,
            mqtt_port => $mqtt_port
        );
    } else {
        $client = Moustique->new(
            ip => $ip,
            port => $port,
            name => "PerlTest-$protocol",
            username => $user,
            password => $pass
        );
    }

    print "1. Client created: " . $client->get_client_name() . "\n";

    # Test variables
    my $received_count = 0;
    my $test_topic = "test/perl/$protocol";

    # Subscribe to test topic
    $client->subscribe($test_topic, sub {
        my ($topic, $message, $from) = @_;
        print "   📨 Received: topic=$topic, message=$message, from=$from\n";
        $received_count++;
    });

    print "2. Subscribed to $test_topic\n";

    # Give subscription time to register
    sleep(1);

    # Publish test message
    my $test_message = "Hello from Perl ($protocol) - " . time();
    $client->publish($test_topic, $test_message, $client->get_client_name());
    print "3. Published message: $test_message\n";

    # Wait for messages
    if ($use_mqtt) {
        print "   Waiting for MQTT messages (10 seconds)...\n";
        for (my $i = 0; $i < 100; $i++) {
            $client->tick();
            sleep(0.1);
        }
    } else {
        print "   Polling for HTTP messages (10 seconds)...\n";
        for (my $i = 0; $i < 20; $i++) {
            $client->tick();
            sleep(0.5);
        }
    }

    # Check results
    print "4. Messages received: $received_count\n";

    # Disconnect if MQTT
    if ($use_mqtt) {
        $client->disconnect();
        print "5. Disconnected from MQTT\n";
    }

    my $success = ($received_count > 0);
    print "   Result: " . ($success ? "✓ PASS" : "✗ FAIL") . "\n";

    return $success;
}
