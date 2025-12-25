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
    $server_url="http://" . ($params{ip} ||"cloud.moustique.xyz");
    $server_port=$params{port} || "33334";

    # Authentication: use provided credentials, fall back to global, or use undef for public
    $self->{username} = $params{username} // $GLOBAL_USERNAME;
    $self->{password} = $params{password} // $GLOBAL_PASSWORD;

    $self->{callbacks}={};
    #    $self->{consumers}={};
    $self->{system_callbacks}={};
    $self->{ua} = LWP::UserAgent->new(
        timeout       => 15,
        keep_alive    => 10,
        agent         => "Moustique/2.0",
    );
    $self->initialize();
    return $self;
}

sub initialize {
  my $self = shift;
  $self->{server_ip}=$server_ip;
  $self->{server_port}=$server_port;
  #  my %callbacks = ();
  #  my %system_callbacks = ();
  #$system_callbacks{"/server/action/resubscribe"}=\&{ sub { resubscribe() } };
  #$self->{system_callbacks}{"/server/action/resubscribe"}=\&resubscribe;
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
  my $post_url=$self->{server_ip} . ":" . $self->{server_port} . "/POST";

  my $form = $self->add_auth({
    topic => enc($topic),
    message => enc($message),
    updated_time => enc(time),
    updated_nicedatetime => enc(get_nicedatetime()),
    from => enc($from)
  });

  my $response = $mua->post( "http://" . $post_url, $form);
  while(!$response->is_success && $retries < $POST_RETRIES) {
    $response = $mua->post( "http://" . $post_url, $form);
    $retries+=1;
  }
  unless($response->is_success) {
    warn "Moustique->publish FAILED, $package:$filename:$subroutine:$line $topic response code: " . $response->code;
  }
  return $response->code;
}

# Not threaded, class sub
sub publish_nothread {
  my ($ip, $port, $topic, $message, $from, $username, $password) = @_;
  my $retries = 0;
  my $post_url=$ip . ":" . $port . "/POST";

  # Use provided credentials, fall back to globals, or use undef for public
  $username = $GLOBAL_USERNAME unless defined $username;
  $password = $GLOBAL_PASSWORD unless defined $password;

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

  my $response = $gua->post( "http://" . $post_url, $form);
  while(!$response->is_success && $retries < $POST_RETRIES) { #Forsok $POST_RETRIES ganger eller tills $response->is_success
    $response = $gua->post( "http://" . $post_url, $form);
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
sub publish_nothread_put {
  my ($ip, $port, $topic, $message, $from, $username, $password) = @_;
  my $post_url=$ip . ":" . $port . "/PUTVAL";

  # Use provided credentials, fall back to globals, or use undef for public
  $username = $GLOBAL_USERNAME unless defined $username;
  $password = $GLOBAL_PASSWORD unless defined $password;

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

  $gua->put( "http://" . $post_url, $form);
}

#Consumer can be one Scene out of x scenes using the same Moustique-client. To stop each scene from ticking every second causing x ticks a second, the consumers are put in a hash and the tick-sub returns x*1 so that each consumer sleeps x seconds between pickups, resulting in 1 tick per second in total.
sub subscribe {
  my ($self, $topic, $callback, $consumer) = @_;
  #my $ua      = LWP::UserAgent->new(timeout=>5);
  my $form = $self->add_auth({
    topic => enc($topic),
    client => enc($self->{name})
  });

  my $response = $gua->post( $server_url.":".$server_port ."/SUBSCRIBE", $form );
  warn ("$self->{name} subscrbar pa $topic");
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
     my $form = $self->add_auth({
       topic => enc($topic),
       client => enc($self->{name})
     });
     my $response = $gua->post($server_url.":".$server_port ."/SUBSCRIBE" , $form );
  }
  publish_nothread("localhost", "33334", "/mushroom/logs/moustique_lib/DEBUG", "$self->{name} Resubscribed all subscriptions", $self->{name}, $self->{username}, $self->{password});
}

sub tick {
  my ($self, $consumer) = @_;
  $self->pickup();
  #  return scalar keys %{ $self->{consumers} } || 1;
  return 1;
}

sub getval {
  my ($ip, $port, $valname, $username, $password) = @_;
  #my $ua      = LWP::UserAgent->new(timeout=>5);
  my $retries = 0;
  my $post_url="http://" . $ip . ":" . $port . "/GETVAL";
  my $retval=undef;
  my ( $package, $filename, $line, $subroutine ) = caller(2);

  # Use provided credentials, fall back to globals, or use undef for public
  $username = $GLOBAL_USERNAME unless defined $username;
  $password = $GLOBAL_PASSWORD unless defined $password;

  my %form;
  $form{'client'}=enc($name);
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
  my $post_url="http://$self->{server_ip}:$self->{server_port}/GETVAL";
  my $retval=undef;
  my $retries = 0;

  my $form = $self->add_auth({
    client => enc($name),
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
  my ($ip, $port, $regex, $username, $password) = @_;
  #my $ua      = LWP::UserAgent->new(timeout=>5);
  my $post_url="http://" . $ip . ":" . $port . "/GETVALSBYREGEX";
  my $matched;
  my @matched_values;

  # Use provided credentials, fall back to globals, or use undef for public
  $username = $GLOBAL_USERNAME unless defined $username;
  $password = $GLOBAL_PASSWORD unless defined $password;

  my %form;
  $form{'client'}=enc($name);
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
  my ($ip, $port, $topic, $message, $from, $username, $password) = @_;
  publish_nothread_put($ip, $port, $topic, $message, $from, $username, $password);
}

sub get_version {
  my ($self,$ip,$port,$pwd) = @_;
  return $self->get_($ip,$port,$pwd,"/VERSION");
}

sub getversion {
  my ($ip,$port,$pwd) = @_;
  return get($ip,$port,$pwd,"/VERSION");
}

sub getfileversion {
  my ($ip,$port,$pwd) = @_;
  return get($ip,$port,$pwd,"/FILEVERSION");
}

sub getstats {
  my ($ip,$port,$pwd) = @_;
  return get($ip,$port,$pwd,"/STATS");
}

sub getclients {
  my ($ip,$port,$pwd) = @_;
  return get($ip,$port,$pwd,"/CLIENTS");
}

sub getposters {
  my ($ip,$port,$pwd) = @_;
  return get($ip,$port,$pwd,"/POSTERS");
}

sub gettopics {
  my ($ip,$port,$pwd) = @_;
  return get($ip,$port,$pwd,"/TOPICS");
}

sub getpeerhosts {
  my ($ip,$port,$pwd) = @_;
  return get($ip,$port,$pwd,"/PEERHOSTS");
}

sub getcrooks {
  my ($ip,$port,$pwd) = @_;
  return get($ip,$port,$pwd,"/CROOKS");
}

sub get_ {
  my ($self,$ip,$port,$pwd,$endpoint,$retries) = @_;
  $retries ||= 0;
  my $mua      =  $self->{ua}; #LWP::UserAgent->new(timeout=>8);
  my $post_url="http://" . $ip . ":" . $port . "/$endpoint";
  my $retval=undef;

  my %form;
  $form{'client'}=enc($name);
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

sub get {
  my ($ip,$port,$pwd,$endpoint,$retries) = @_;
  $retries ||= 0;
  #my $ua      = LWP::UserAgent->new(timeout=>8);
  my $post_url="http://" . $ip . ":" . $port . "/$endpoint";
  my $retval=undef;

  my %form;
  $form{'client'}=enc($name);
  $form{'pwd'}=enc($pwd);
  my $response = $gua->post( $post_url, \%form );
  if($response->is_success) {
    my $respcont = dec($response->content);
    $retval = decode_json($respcont);
  } elsif ($response->code() eq "401") {
    print "Vanligen ange pwd.\n";
  } elsif ($retries < 5) {
    get($ip,$port,$pwd,$endpoint,$retries+1);
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

  my $response = $mua->post( $server_url.":".$server_port ."/PICKUP", $form );
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

1;
