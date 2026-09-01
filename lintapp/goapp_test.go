package main

import "testing"

func Test(t *testing.T) {
	if 1 == 2 {
		t.Fatal("bad mojo")
	}
}
