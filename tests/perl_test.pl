#!/usr/bin/perl
# Perl client test for Moustique
# Usage:
#   perl perl_test.pl public <host> <port>
#   perl perl_test.pl auth <host> <port> <username> <password>

use strict;
use warnings;
use lib '../clients/perl';
use Moustique;

sub test_public {
    my ($host, $port) = @_;

    eval {
        my $response = Moustique::publish_nothread(
            $host,
            $port,
            "/test/perl/public",
            "Hello from Perl public!",
            "perl-test"
        );

        if ($response == 200) {
            print "✓ Perl public publish successful\n";
            return 1;
        } else {
            print "✗ Perl public publish failed: HTTP $response\n";
            return 0;
        }
    };

    if ($@) {
        print "✗ Perl public publish failed: $@\n";
        return 0;
    }
}

sub test_auth {
    my ($host, $port, $username, $password) = @_;

    # Set global credentials
    $Moustique::GLOBAL_USERNAME = $username;
    $Moustique::GLOBAL_PASSWORD = $password;

    eval {
        my $response = Moustique::publish_nothread(
            $host,
            $port,
            "/test/perl/auth",
            "Hello from Perl auth!",
            "perl-test"
        );

        if ($response == 200) {
            print "✓ Perl authenticated publish successful\n";
            return 1;
        } else {
            print "✗ Perl authenticated publish failed: HTTP $response\n";
            return 0;
        }
    };

    if ($@) {
        print "✗ Perl authenticated publish failed: $@\n";
        return 0;
    }
}

# Main
if (@ARGV < 3) {
    print "Usage: perl perl_test.pl <public|auth> <host> <port> [username] [password]\n";
    exit 1;
}

my $mode = $ARGV[0];
my $host = $ARGV[1];
my $port = $ARGV[2];

my $success;

if ($mode eq "public") {
    $success = test_public($host, $port);
} elsif ($mode eq "auth") {
    if (@ARGV < 5) {
        print "Auth mode requires username and password\n";
        exit 1;
    }
    my $username = $ARGV[3];
    my $password = $ARGV[4];
    $success = test_auth($host, $port, $username, $password);
} else {
    print "Unknown mode: $mode\n";
    exit 1;
}

exit($success ? 0 : 1);
