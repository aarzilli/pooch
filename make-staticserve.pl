#!/usr/bin/env perl
use warnings;
use strict;

use File::Basename;

# This program is distributed under the terms of GPLv3
# Copyright 2010, Alessandro Arzilli

use MIME::Base64;

print "package pooch\n\n";
print "var FILES map[string]string = map[string]string{\n";

my %sums = ();

for my $curarg (@ARGV) {
    next if -d $curarg;
    open my $in, '<', $curarg or die "Couldn't read $curarg: $!";
    my $text = do { local $/; <$in> };
    close $in;

    #my $name = basename($curarg);
    my ($first_piece, $name) = split '/', $curarg, 2;

    print STDERR "Adding $name\n";

    print "\t\"$name\": \"".encode_base64($text, "")."\",\n";

    my $x = do { chomp(my $ret = `md5sum $curarg`); my @v = split / /, $ret; $v[0]};
    #my $x = `md5sum $curarg`;

    $sums{$name} = $x;
}

print "}\n";
print "\n";

print "var SUMS map[string]string = map[string]string{\n";
for my $curarg (keys %sums) {
    print "\t\"$curarg\": \"$sums{$curarg}\",\n";
}

print "}\n";
print "\n";

