#!/usr/bin/perl
use Data::Dumper;


print("Testar med $ARGV[0]\n");
my @monster = @{explode_topic($ARGV[0])};
print Dumper \@monster;

sub explode_topic {
  my ($topic) = @_;

  my @patterns;

  my @sections = split('/', $topic);
  my $pattern = '';
  my $insprangt_wildcard = '';

  for (my $i = $#sections; $i >= 1; $i--) {
      $pattern = join("/", @sections[0..$i-1]) . "/" . join('/', map { $_ eq $sections[$i] ? $_ : '+' } @sections[$i..$#sections]);
      $insprangt_wildcard = (join("/", @sections[0..$i-2]) . "/+/" . join("/", @sections[$i..$#sections])) if($i>2 && $i <= $#sections);
      push @patterns, $pattern;
      push @patterns, $insprangt_wildcard if($i>2 && $i <= $#sections);
  }

  return \@patterns;
}
