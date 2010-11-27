#!/usr/bin/env perl
use warnings;
use strict;

open my $in, './pooch search -t|' or die "Couldn't open input: $!";
my @lines = <$in>;
close $in;

for (@lines) {
    chomp;
    my ($id, @rest) = split /\t/, $_;
    next unless $id =~ m{/};

    system "./pooch rename $id";
}

