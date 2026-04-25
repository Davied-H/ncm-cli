package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseSongIDs(t *testing.T) {
	ids, err := parseSongIDs([]string{"1", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids = %#v", ids)
	}
	if _, err := parseSongIDs([]string{"0"}); err == nil {
		t.Fatal("expected invalid id error")
	}
}

func TestCleanTags(t *testing.T) {
	tags := cleanTags([]string{"华语, 流行", " 摇滚 "})
	if strings.Join(tags, "|") != "华语|流行|摇滚" {
		t.Fatalf("tags = %#v", tags)
	}
}

func TestConfirmExpected(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("yes\n"))
	cmd.SetErr(&bytes.Buffer{})
	if err := confirmExpected(cmd, false, "prompt", "yes"); err != nil {
		t.Fatal(err)
	}

	cmd = &cobra.Command{}
	cmd.SetIn(strings.NewReader("no\n"))
	cmd.SetErr(&bytes.Buffer{})
	if err := confirmExpected(cmd, false, "prompt", "yes"); err == nil {
		t.Fatal("expected cancellation")
	}

	cmd = &cobra.Command{}
	cmd.SetIn(strings.NewReader(""))
	cmd.SetErr(&bytes.Buffer{})
	if err := confirmExpected(cmd, true, "prompt", "yes"); err != nil {
		t.Fatal(err)
	}
}
