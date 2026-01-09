package Moustique;

use strict;
use warnings;
no warnings qw( experimental::smartmatch );
use Sys::Hostname;
use JSON;
use LWP::UserAgent;
use MIME::Base64;
use Data::Dumper;
use POSIX qw(getpid);

# Optional MQTT support
# Note: Net::MQTT::Simple has limitations - it requires blocking event loop
# For production MQTT use in Perl, consider Net::MQTT or AnyEvent::MQTT
# For now, MQTT is disabled in Perl client - use HTTP instead
my $MQTT_AVAILABLE = 0;  # Disabled due to Net::MQTT::Simple limitations
# eval {
#     require Net::MQTT::Simple;
#     $MQTT_AVAILABLE = 1;
# };
if (!$MQTT_AVAILABLE) {
    # MQTT not available, will use HTTP only
}

my $gua = LWP::UserAgent->new(
        timeout       => 15,
        keep_alive    => 10,
);
my $json = JSON::XS->new->allow_nonref;
my $sent_cnt=0;
my $url = 'http://moustique.host:33334/POST';
my $server_ip;
my $server_url;
my $server_port;
my $pickup_intensity=1;
my $name="NONAME";
my $POST_RETRIES=5;

# Global authentication credentials
# Set these if you want default authentication for all operations
# Can be overridden by passing username/password to new()
our $GLOBAL_USERNAME = undef;
our $GLOBAL_PASSWORD = undef;

# Global TLS setting for class methods
# Set to 1 to use HTTPS instead of HTTP
our $GLOBAL_USE_TLS = 0;

# Helper to get URL scheme based on TLS setting
sub _get_scheme {
  my ($use_tls) = @_;
  $use_tls = $GLOBAL_USE_TLS unless defined $use_tls;
  return $use_tls ? "https" : "http";
}

sub new {
    my $class = shift;
    my %params = @_;
    my $self = bless {}, $class;
    my $pid = getpid();         # Hämta processens id (PID)
    $name=hostname;
    $name = $name . "-" . $params{name} if(defined $params{name} && "" ne $params{name});
    $name = $name . "-" . int(rand(100)) . "-$pid";
    $self->{name} = $name;
    $server_ip="". ($params{ip} ||"cloud.moustique.xyz");
    my $scheme = $params{use_tls} ? "https" : "http";
    $server_url=$scheme . "://" . ($params{ip} ||"cloud.moustique.xyz");
    $server_port=$params{port} || "33334";

    # TLS support
    $self->{use_tls} = $params{use_tls} || 0;

    # Authentication: use provided credentials, fall back to global, or use undef for public
    $self->{username} = $params{username} // $GLOBAL_USERNAME;
    $self->{password} = $params{password} // $GLOBAL_PASSWORD;

    # MQTT support
    $self->{use_mqtt} = $params{use_mqtt} && $MQTT_AVAILABLE;
    $self->{mqtt_port} = $params{mqtt_port} || 1883;
    $self->{mqtt_client} = undef;
    $self->{mqtt_connected} = 0;

    if ($params{use_mqtt} && !$MQTT_AVAILABLE) {
        warn "MQTT requested but not supported in Perl client (library limitations). Falling back to HTTP.\n";
        warn "Note: Net::MQTT::Simple requires blocking event loop incompatible with tick() pattern.\n";
        warn "For production MQTT, consider using other Moustique clients (Python, JavaScript, Go, Java).\n";
    }

    $self->{callbacks}={};
    #    $self->{consumers}={};
    $self->{system_callbacks}={};
    $self->{ua} = LWP::UserAgent->new(
        timeout       => 15,
        keep_alive    => 10,
        agent         => "Moustique/2.0",
    );
    $self->initialize();

    if ($self->{use_mqtt}) {
        $self->_init_mqtt();
    }

    return $self;
}

sub initialize {
  my $self = shift;
  $self->{server_ip}=$server_ip;
  my $scheme = $self->{use_tls} ? "https" : "http";
  $self->{server_url}=$scheme . "://" . $server_ip;
  $self->{server_port}=$server_port;
  $self->{system_callbacks}{"/server/action/resubscribe"}=sub { $self->resubscribe(@_) };
}

sub add_auth {
  my ($self, $form) = @_;
  if (defined $self->{username} && defined $self->{password}) {
    $form->{username} = enc($self->{username});
    $form->{password} = enc($self->{password});
  }
  return $form;
}

sub publish {
  my ($self, $topic, $message,$from) = @_;
  my $retries = 0;
  my $mua      = $self->{ua};
  my ( $package, $filename, $line, $subroutine ) = caller(1);
  my $post_url=$self->{server_url} . ":" . $self->{server_port} . "/POST";

  my $form = $self->add_auth({
    topic => enc($topic),
    message => enc($message),
    updated_time => enc(time),
    updated_nicedatetime => enc(get_nicedatetime()),
    from => enc($from)
  });

  my $response = $mua->post( $post_url, $form);
  while(!$response->is_success && $retries < $POST_RETRIES) {
    $response = $mua->post( $post_url, $form);
    $retries+=1;
  }
  unless($response->is_success) {
    warn "Moustique->publish FAILED, $package:$filename:$subroutine:$line $topic response code: " . $response->code;
  }
  return $response->code;
}

# Not threaded, class sub
# Usage: publish_nothread($ip, $port, $topic, $message, $from, $username, $password, $client_name, $use_tls)
sub publish_nothread {
  my ($ip, $port, $topic, $message, $from, $username, $password, $client_name, $use_tls) = @_;
  my $retries = 0;
  my $scheme = _get_scheme($use_tls);
  my $post_url=$scheme . "://" . $ip . ":" . $port . "/POST";

  # Use provided credentials, fall back to globals, or use undef for public
  $username = $GLOBAL_USERNAME unless defined $username;
  $password = $GLOBAL_PASSWORD unless defined $password;
  $client_name = $name unless defined $client_name;  # Fall back to global if not provided

  my $form = {
    topic => enc($topic),
    message => enc($message),
    updated_time => enc(time),
    updated_nicedatetime => enc(get_nicedatetime()),
    from => enc($from)
  };

  # Add auth if credentials are available
  if (defined $username && defined $password) {
    $form->{username} = enc($username);
    $form->{password} = enc($password);
  }

  my $response = $gua->post( $post_url, $form);
  while(!$response->is_success && $retries < $POST_RETRIES) { #Forsok $POST_RETRIES ganger eller tills $response->is_success
    $response = $gua->post( $post_url, $form);
    $retries+=1;
    warn "Retrying publish [$retries/$POST_RETRIES]";
  }
  unless($response->is_success) {
    warn "Moustique::publish_nothread FAILED, response code: " . $response->code;
  }
  return $response->code;
}
#
# Not threaded, class sub
# Usage: publish_nothread_put($ip, $port, $topic, $message, $from, $username, $password, $client_name, $use_tls)
sub publish_nothread_put {
  my ($ip, $port, $topic, $message, $from, $username, $password, $client_name, $use_tls) = @_;
  my $scheme = _get_scheme($use_tls);
  my $post_url=$scheme . "://" . $ip . ":" . $port . "/PUTVAL";

  # Use provided credentials, fall back to globals, or use undef for public
  $username = $GLOBAL_USERNAME unless defined $username;
  $password = $GLOBAL_PASSWORD unless defined $password;
  $client_name = $name unless defined $client_name;  # Fall back to global if not provided

  my $form = {
    valname => enc($topic),
    val => enc($message),
    updated_time => enc(time),
    updated_nicedatetime => enc(get_nicedatetime()),
    from => enc($from)
  };

  # Add auth if credentials are available
  if (defined $username && defined $password) {
    $form->{username} = enc($username);
    $form->{password} = enc($password);
  }

  $gua->put( $post_url, $form);
}

sub subscribe {
  my ($self, $topic, $callback, $consumer) = @_;

  # Store callback
  my %callbacks= %{ $self->{callbacks} };
  unless($callbacks{$topic}) {
    $callbacks{$topic} = ();
  }
  my $exists=0;
  foreach my $cb (@{$callbacks{$topic}}) {
    $exists=1 if($cb == $callback);
    print ("Hittade samma callback $cb for amnet $topic!\n") if($cb == $callback);
  }
  push(@{$callbacks{$topic}}, $callback) unless $exists;
  $self->{callbacks}=\%callbacks;

  # MQTT subscription
  if ($self->{use_mqtt} && $self->{mqtt_connected}) {
    eval {
      # Store the subscription for tick() processing
      $self->{mqtt_subscriptions}{$topic} = 1;

      # Subscribe with callback
      $self->{mqtt_client}->subscribe($topic, sub {
        my ($topic_received, $message) = @_;
        my $msg_topic = $topic_received;
        my $msg_text = $message;
        my $msg_from = '';

        # Try to parse as JSON first (for compatibility)
        # If it fails, treat as plaintext (standard MQTT)
        eval {
          my $msg_obj = decode_json($message);
          if (ref $msg_obj eq 'HASH' && $msg_obj->{message}) {
            $msg_topic = $msg_obj->{topic} || $topic_received;
            $msg_text = $msg_obj->{message};
            $msg_from = $msg_obj->{from} || '';
          }
        };
        # If JSON parse fails, already set to plaintext above

        # Find matching callbacks
        my %callbacks = %{ $self->{callbacks} };
        foreach my $subscribed_topic (keys %callbacks) {
          if ($self->_topic_matches($subscribed_topic, $msg_topic)) {
            my @topic_callbacks = @{$callbacks{$subscribed_topic} || []};
            foreach my $callback (@topic_callbacks) {
              eval {
                $callback->($msg_topic, $msg_text, $msg_from);
              };
              if ($@) {
                warn "Error in callback for topic '$topic_received': $@\n";
              }
            }
          }
        }
      });

      print "✓ Subscribed to $topic via MQTT\n";
      return;
    };
    if ($@) {
      warn "MQTT subscribe failed, falling back to HTTP: $@\n";
      # Fall through to HTTP subscription
    }
  }

  # HTTP subscription
  my $form = $self->add_auth({
    topic => enc($topic),
    client => enc($self->{name})
  });

  my $response = $gua->post( $self->{server_url}.":".$self->{server_port} ."/SUBSCRIBE", $form );
  warn ("$self->{name} subscrbar pa $topic");
}

# Calls subscribe on the server for all subscriptions we have.
# This is triggered by the system message /server/action/resubscribe which is issued by the server as it starts 
# in order to restore any existing clients at a restart.
sub resubscribe {
  my ($self) = @_;
  #my $ua      = LWP::UserAgent->new(timeout=>5);
  my %callbacks=%{ $self->{callbacks} };
  my @subs = keys %callbacks;
  publish_nothread("localhost", "33334", "/mushroom/logs/moustique_lib/DEBUG", "$self->{name} Resubscribing all subscriptions", $self->{name}, $self->{username}, $self->{password}) if scalar @subs > 0;
  foreach my $topic (@subs) {
     print("Resubscribing $topic " . $self->{name} . "\n");

     if ($self->{use_mqtt} && $self->{mqtt_connected}) {
       # MQTT resubscribe
       eval {
         # Re-subscribe with the same pattern as subscribe()
         $self->{mqtt_subscriptions}{$topic} = 1;

         $self->{mqtt_client}->subscribe($topic, sub {
           my ($topic_received, $message) = @_;
           my $msg_topic = $topic_received;
           my $msg_text = $message;
           my $msg_from = '';

           eval {
             my $msg_obj = decode_json($message);
             if (ref $msg_obj eq 'HASH' && $msg_obj->{message}) {
               $msg_topic = $msg_obj->{topic} || $topic_received;
               $msg_text = $msg_obj->{message};
               $msg_from = $msg_obj->{from} || '';
             }
           };

           my %callbacks = %{ $self->{callbacks} };
           foreach my $subscribed_topic (keys %callbacks) {
             if ($self->_topic_matches($subscribed_topic, $msg_topic)) {
               my @topic_callbacks = @{$callbacks{$subscribed_topic} || []};
               foreach my $callback (@topic_callbacks) {
                 eval { $callback->($msg_topic, $msg_text, $msg_from); };
                 warn "Error in callback for topic '$topic_received': $@\n" if $@;
               }
             }
           }
         });

         print "✓ Re-subscribed to $topic via MQTT\n";
       };
       if ($@) {
         warn "Failed to resubscribe to '$topic' via MQTT: $@\n";
       }
     } else {
       # HTTP resubscribe
       my $form = $self->add_auth({
         topic => enc($topic),
         client => enc($self->{name})
       });
       my $response = $gua->post($self->{server_url}.":".$self->{server_port} ."/SUBSCRIBE" , $form );
     }
  }
  publish_nothread("localhost", "33334", "/mushroom/logs/moustique_lib/DEBUG", "$self->{name} Resubscribed all subscriptions", $self->{name}, $self->{username}, $self->{password});
}

sub tick {
  my ($self, $consumer) = @_;

  # HTTP polling (MQTT not supported in Perl client)
  $self->pickup();

  #  return scalar keys %{ $self->{consumers} } || 1;
  return 1;
}

sub getval {
  my ($ip, $port, $valname, $username, $password, $client_name, $use_tls) = @_;
  #my $ua      = LWP::UserAgent->new(timeout=>5);
  my $retries = 0;
  my $scheme = _get_scheme($use_tls);
  my $post_url=$scheme . "://" . $ip . ":" . $port . "/GETVAL";
  my $retval=undef;
  my ( $package, $filename, $line, $subroutine ) = caller(2);

  # Use provided credentials, fall back to globals, or use undef for public
  $username = $GLOBAL_USERNAME unless defined $username;
  $password = $GLOBAL_PASSWORD unless defined $password;
  $client_name = $name unless defined $client_name;  # Fall back to global if not provided

  my %form;
  $form{'client'}=enc($client_name);
  $form{'topic'}=enc($valname);

  # Add auth if credentials are available
  if (defined $username && defined $password) {
    $form{'username'} = enc($username);
    $form{'password'} = enc($password);
  }

  my $response = $gua->post( $post_url, \%form );
  while(!$response->is_success && $response->code != 404 && $retries < $POST_RETRIES) {
    $response = $gua->post( $post_url, \%form );
    $retries+=1;
    warn "Retrying getval [$retries/$POST_RETRIES]";
  }
  if($response->is_success) {
    my $respcont = dec($response->content);
    $retval = decode_json($respcont);
  } elsif($response->code != 404) {
    warn "Moustique::getval FAILED $package:$filename:$subroutine:$line $ip:$port$valname, response code: " . $response->code;
  }
  return $retval;
}

sub get_val {
  my ($self, $valname) = @_;
  my $mua      = $self->{ua}; #/LWP::UserAgent->new(timeout=>5);
  my $scheme = $self->{use_tls} ? "https" : "http";
  my $post_url="$scheme://$self->{server_ip}:$self->{server_port}/GETVAL";
  my $retval=undef;
  my $retries = 0;

  my $form = $self->add_auth({
    client => enc($self->{name}),  # Use instance variable instead of global
    topic => enc($valname)
  });

  my $response = $mua->post( $post_url, $form );
  while(!$response->is_success && $response->code != 404 && $retries < $POST_RETRIES) {
    $response = $mua->post( $post_url, $form );
    $retries+=1;
    warn "Retrying get_val [$retries/$POST_RETRIES]";
  }
  if($response->is_success) {
    my $respcont = dec($response->content);
    $retval = decode_json($respcont);
  } elsif($response->code != 404) {
    warn "Moustique->get_val FAILED, response code: $self->{server_ip}:$self->{server_port}$valname" . $response->code;
  }
  return $retval;
}

sub get_vals_by_regex {
  my ($ip, $port, $regex, $username, $password, $client_name, $use_tls) = @_;
  #my $ua      = LWP::UserAgent->new(timeout=>5);
  my $scheme = _get_scheme($use_tls);
  my $post_url=$scheme . "://" . $ip . ":" . $port . "/GETVALSBYREGEX";
  my $matched;
  my @matched_values;

  # Use provided credentials, fall back to globals, or use undef for public
  $username = $GLOBAL_USERNAME unless defined $username;
  $password = $GLOBAL_PASSWORD unless defined $password;
  $client_name = $name unless defined $client_name;  # Fall back to global if not provided

  my %form;
  $form{'client'}=enc($client_name);
  $form{'topic'}=enc($regex);

  # Add auth if credentials are available
  if (defined $username && defined $password) {
    $form{'username'} = enc($username);
    $form{'password'} = enc($password);
  }

  my $response = $gua->post( $post_url, \%form );
  if($response->is_success) {
    my $respcont = dec($response->content);
    $matched = decode_json($respcont);
    if(scalar keys %$matched > 0) {
      @matched_values = values %$matched;
    }
  }
  return \@matched_values;
}

sub putval {
  my ($ip, $port, $topic, $message, $from, $username, $password, $client_name, $use_tls) = @_;
  publish_nothread_put($ip, $port, $topic, $message, $from, $username, $password, $client_name, $use_tls);
}

sub get_version {
  my ($self,$ip,$port,$pwd) = @_;
  return $self->get_($ip,$port,$pwd,"/VERSION");
}

# Usage: getversion($ip, $port, $pwd, $client_name)
sub getversion {
  my ($ip,$port,$pwd,$client_name) = @_;
  return get($ip,$port,$pwd,"/VERSION",0,$client_name);
}

# Usage: getfileversion($ip, $port, $pwd, $client_name)
sub getfileversion {
  my ($ip,$port,$pwd,$client_name) = @_;
  return get($ip,$port,$pwd,"/FILEVERSION",0,$client_name);
}

# Usage: getstats($ip, $port, $pwd, $client_name)
sub getstats {
  my ($ip,$port,$pwd,$client_name) = @_;
  return get($ip,$port,$pwd,"/STATS",0,$client_name);
}

# Usage: getclients($ip, $port, $pwd, $client_name)
sub getclients {
  my ($ip,$port,$pwd,$client_name) = @_;
  return get($ip,$port,$pwd,"/CLIENTS",0,$client_name);
}

# Usage: getposters($ip, $port, $pwd, $client_name)
sub getposters {
  my ($ip,$port,$pwd,$client_name) = @_;
  return get($ip,$port,$pwd,"/POSTERS",0,$client_name);
}

# Usage: gettopics($ip, $port, $pwd, $client_name)
sub gettopics {
  my ($ip,$port,$pwd,$client_name) = @_;
  return get($ip,$port,$pwd,"/TOPICS",0,$client_name);
}

# Usage: getpeerhosts($ip, $port, $pwd, $client_name)
sub getpeerhosts {
  my ($ip,$port,$pwd,$client_name) = @_;
  return get($ip,$port,$pwd,"/PEERHOSTS",0,$client_name);
}

# Usage: getcrooks($ip, $port, $pwd, $client_name)
sub getcrooks {
  my ($ip,$port,$pwd,$client_name) = @_;
  return get($ip,$port,$pwd,"/CROOKS",0,$client_name);
}

sub get_ {
  my ($self,$ip,$port,$pwd,$endpoint,$retries) = @_;
  $retries ||= 0;
  my $mua      =  $self->{ua}; #LWP::UserAgent->new(timeout=>8);
  my $scheme = $self->{use_tls} ? "https" : "http";
  my $post_url=$scheme . "://" . $ip . ":" . $port . "/$endpoint";
  my $retval=undef;

  my %form;
  $form{'client'}=enc($self->{name});  # Use instance variable instead of global
  $form{'pwd'}=enc($pwd);
  $form{'time'}=enc(time);
  my $response = $mua->post( $post_url, \%form );
  if($response->is_success) {
    my $respcont = dec($response->content);
    $retval = decode_json($respcont);
  } elsif ($response->code() eq "401") {
    print "Vanligen ange pwd.\n";
  } elsif ($retries < $POST_RETRIES) {
    warn "get_ failed, retrying [$retries/$POST_RETRIES]";
    $self->get_($ip,$port,$pwd,$endpoint,$retries+1);
  } else {
    warn $response->status_line()."\n";
  }
  return $retval;
}

# Usage: get($ip, $port, $pwd, $endpoint, $retries, $client_name, $use_tls)
sub get {
  my ($ip,$port,$pwd,$endpoint,$retries,$client_name,$use_tls) = @_;
  $retries ||= 0;
  $client_name = $name unless defined $client_name;  # Fall back to global if not provided

  #my $ua      = LWP::UserAgent->new(timeout=>8);
  my $scheme = _get_scheme($use_tls);
  my $post_url=$scheme . "://" . $ip . ":" . $port . "/$endpoint";
  my $retval=undef;

  my %form;
  $form{'client'}=enc($client_name);
  $form{'pwd'}=enc($pwd);
  my $response = $gua->post( $post_url, \%form );
  if($response->is_success) {
    my $respcont = dec($response->content);
    $retval = decode_json($respcont);
  } elsif ($response->code() eq "401") {
    print "Vanligen ange pwd.\n";
  } elsif ($retries < 5) {
    get($ip,$port,$pwd,$endpoint,$retries+1,$client_name,$use_tls);
  } else {
    warn $response->status_line()."\n";
  }
  return $retval;
}

sub pickup {
  my $self = shift;
  my $mua      = $self->{ua}; #LWP::UserAgent->new(timeout=>5);

  my %callbacks=%{ $self->{callbacks} };
  my %system_callbacks=%{ $self->{system_callbacks} };

  my $form = $self->add_auth({
    client => enc($self->{name})
  });

  my $response = $mua->post( $self->{server_url}.":".$self->{server_port} ."/PICKUP", $form );
  if($response->is_success) {
    my $respcont=dec($response->content);
    $response = $json->decode($respcont);
    unless(!$response) {
      foreach my $subscribed_topic (keys %{$response}) {
        my @messages = @{$response->{$subscribed_topic}};
        foreach my $message (@messages) {
	  my $topic=$message->{topic};
	  unless(!$callbacks{$subscribed_topic} ){
            my @topic_callbacks=@{$callbacks{$subscribed_topic} || ()};
	    foreach my $callback (@topic_callbacks) {
	      $callback->($topic,$message->{message},$message->{from});
	    }
          } else {
            unless(!$system_callbacks{$subscribed_topic} ) {
              my $callback = $system_callbacks{$subscribed_topic};
	      warn "Got System Message";
	      $callback->($topic,$message->{message});
            } else {
	      warn "Got $topic " .$message->{message};
	    }
          }
        }
      }
    }
  }
}

sub get_nicedatetime {
  my ($second, $minute, $hour, $dayOfMonth, $month, $yearOffset, $dayOfWeek, $dayOfYear, $daylightSavings) = localtime();
  my $year = 1900 + $yearOffset;
  $month += 1;
  $month = "0$month" if($month < 10);
  $minute = "0$minute" if($minute < 10);
  $second = "0$second" if($second < 10);
  $dayOfMonth= "0$dayOfMonth" if($dayOfMonth < 10);
  my $nicedate = $year . "-" . $month . "-" . $dayOfMonth . " " . $hour . ":" . $minute . ":" . $second;
  return $nicedate;
}

sub get_client_name {
  my $self = shift;
  return $self->{name};
}

# Set the client's "AboutMe" description.
# This description can be viewed in the admin panel and helps identify
# what this client does (e.g., "Cron job on server X that sends sensor data").
sub set_about_me {
  my ($self, $about_me) = @_;
  my $mua = $self->{ua};
  my $scheme = $self->{use_tls} ? "https" : "http";
  my $post_url = "$scheme://$self->{server_ip}:$self->{server_port}/SET_ABOUT_ME";

  my $form = $self->add_auth({
    client => enc($self->{name}),
    about_me => enc($about_me),
    type => enc("client")
  });

  my $response = $mua->post($post_url, $form);
  unless ($response->is_success) {
    warn "set_about_me failed: " . $response->code . " " . $response->status_line;
  }
  return $response->is_success;
}

sub enc {
 my ($plaintext) = @_;
 my $encoded;
 if(defined $plaintext) {
   # Must match server encoding: ROT13 first, then Base64
   my $rot13_text = $plaintext;
   $rot13_text =~ tr/A-Za-z/N-ZA-Mn-za-m/;
   $encoded = encode_base64($rot13_text, '') ;  # '' prevents newlines
 }
 return $encoded;
}

sub dec {
  my ($encoded) = @_;
  # Reverse of encode: Base64 decode first, then ROT13
  my $decoded = decode_base64($encoded);
  $decoded =~ tr/A-Za-z/N-ZA-Mn-za-m/;
  return $decoded;
}

# MQTT Support Methods

sub _init_mqtt {
  my ($self) = @_;

  eval {
    my $broker = "$self->{server_ip}:$self->{mqtt_port}";

    # Create MQTT connection
    $self->{mqtt_client} = Net::MQTT::Simple->new($broker);

    # Note: Net::MQTT::Simple doesn't support username/password authentication
    # For authenticated MQTT, users should configure their MQTT broker to allow
    # unauthenticated access or use a different Perl MQTT library

    # Store MQTT subscriptions for later processing
    $self->{mqtt_subscriptions} = {};

    $self->{mqtt_connected} = 1;
    print "✓ Connected to MQTT broker at $broker\n";

  };
  if ($@) {
    warn "MQTT connection failed: $@\n";
    warn "Falling back to HTTP mode\n";
    $self->{use_mqtt} = 0;
  }
}

sub _process_mqtt_messages {
  my ($self) = @_;

  return unless $self->{use_mqtt} && $self->{mqtt_connected};

  # Net::MQTT::Simple handles message processing in background
  # The callbacks registered during subscribe() are called automatically
  # This method is here for future enhancements
}

sub _topic_matches {
  my ($self, $pattern, $topic) = @_;

  # Simple MQTT wildcard matching (+ for single level, # for multi-level)
  my @pattern_parts = split('/', $pattern);
  my @topic_parts = split('/', $topic);

  if (scalar @pattern_parts > scalar @topic_parts && $pattern_parts[-1] ne '#') {
    return 0;
  }

  for (my $i = 0; $i < scalar @pattern_parts; $i++) {
    if ($pattern_parts[$i] eq '#') {
      return 1;  # Match everything after
    }
    return 0 if $i >= scalar @topic_parts;
    next if $pattern_parts[$i] eq '+';  # Match single level
    return 0 if $pattern_parts[$i] ne $topic_parts[$i];
  }

  return scalar @pattern_parts == scalar @topic_parts ||
         (scalar @pattern_parts > 0 && $pattern_parts[-1] eq '#');
}

sub disconnect {
  my ($self) = @_;

  if ($self->{mqtt_client} && $self->{mqtt_connected}) {
    eval {
      $self->{mqtt_client}->disconnect();
      $self->{mqtt_connected} = 0;
      print "MQTT client disconnected\n";
    };
    if ($@) {
      warn "Error disconnecting MQTT client: $@\n";
    }
  }
}

1;
