#!/usr/bin/env perl
use warnings;
use strict;

use MIME::Base64;

print "package main\n\n";
print "var FILES map[string]string = map[string]string{\n";

for my $curarg (@ARGV) {

    open my $in, '<', $curarg or die "Couldn't read $curarg: $!";
    my $text = do { local $/; <$in> };
    close $in;

    print "\t\"$curarg\": \"".encode_base64($text, "")."\",\n";
}

print "}\n";
print "\n";
