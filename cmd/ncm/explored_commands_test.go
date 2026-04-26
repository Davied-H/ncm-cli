package main

import (
	"bytes"
	"strings"
	"testing"

	"ncm-cli/internal/ncm"
)

func TestPlayerURLLevelValidation(t *testing.T) {
	if !validPlayerLevel("exhigh") || !validPlayerLevel("jymaster") {
		t.Fatal("expected known levels to be valid")
	}
	if validPlayerLevel("unknown") {
		t.Fatal("expected unknown level to be invalid")
	}

	cmd := newURLCmd(&rootOptions{})
	cmd.SetArgs([]string{"210049", "--level", "unknown"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--level 不支持") {
		t.Fatalf("err = %v", err)
	}
}

func TestRecordItemsDefaultWeekAndAll(t *testing.T) {
	res := &ncm.PlayRecordResponse{
		WeekData: []ncm.PlayRecordItem{{PlayCount: 1}},
		AllData:  []ncm.PlayRecordItem{{PlayCount: 2}},
	}
	if got := recordItems(res, false); len(got) != 1 || got[0].PlayCount != 1 {
		t.Fatalf("default records = %#v", got)
	}
	if got := recordItems(res, true); len(got) != 1 || got[0].PlayCount != 2 {
		t.Fatalf("all records = %#v", got)
	}
}

func TestRecordFlagValidation(t *testing.T) {
	cmd := newRecordCmd(&rootOptions{})
	cmd.SetArgs([]string{"--week", "--all"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--week 和 --all 不能同时使用") {
		t.Fatalf("err = %v", err)
	}

	cmd = newRecordCmd(&rootOptions{})
	cmd.SetArgs([]string{"--limit", "0"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--limit 必须大于 0") {
		t.Fatalf("err = %v", err)
	}
}

func TestTableLimits(t *testing.T) {
	songs := []ncm.Song{{ID: 1}, {ID: 2}}
	if got := limitSongs(songs, 1); len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("songs = %#v", got)
	}
	records := []ncm.PlayRecordItem{{PlayCount: 1}, {PlayCount: 2}}
	if got := limitRecords(records, 1); len(got) != 1 || got[0].PlayCount != 1 {
		t.Fatalf("records = %#v", got)
	}
}
