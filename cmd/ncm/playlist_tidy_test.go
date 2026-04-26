package main

import (
	"bytes"
	"strings"
	"testing"

	"ncm-cli/internal/ncm"
)

func TestFilterTidySongs(t *testing.T) {
	songs := []tidySong{
		{ID: 1, Name: "晴天", Artists: "周杰伦", Album: "叶惠美", Playable: true},
		{ID: 2, Name: "修炼爱情", Artists: "林俊杰", Album: "因你而在", Playable: false},
	}

	got := filterTidySongs(songs, tidyFilter{Artist: "周", Playable: "yes"})
	if len(got) != 1 || got[0].ID != 1 || !strings.Contains(got[0].Reason, "artist contains 周") {
		t.Fatalf("artist filter = %#v", got)
	}

	got = filterTidySongs(songs, tidyFilter{Album: "因你", Name: "爱情", Playable: "no"})
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("album/name/playable filter = %#v", got)
	}
}

func TestValidateTidyFilter(t *testing.T) {
	if err := validateTidyFilter(tidyFilter{}); err == nil {
		t.Fatal("expected empty filter error")
	}
	if err := validateTidyFilter(tidyFilter{Playable: "maybe"}); err == nil {
		t.Fatal("expected playable validation error")
	}
	if err := validateTidyFilter(tidyFilter{Name: "晴天", Playable: "yes"}); err != nil {
		t.Fatal(err)
	}
}

func TestDiffTidySongs(t *testing.T) {
	source := ncm.Playlist{ID: 10, Name: "source"}
	target := ncm.Playlist{ID: 20, Name: "target"}
	result := diffTidySongs(source, []tidySong{{ID: 1}, {ID: 2}}, target, []tidySong{{ID: 2}, {ID: 3}})

	if result.Summary.Intersection != 1 || result.Summary.OnlySource != 1 || result.Summary.OnlyTarget != 1 {
		t.Fatalf("summary = %#v", result.Summary)
	}
	if result.Intersection[0].ID != 2 || result.OnlySource[0].ID != 1 || result.OnlyTarget[0].ID != 3 {
		t.Fatalf("diff = %#v", result)
	}
}

func TestDuplicateTidySongs(t *testing.T) {
	groups := duplicateTidySongs([]tidySong{{ID: 1, Name: "a"}, {ID: 2}, {ID: 1, Name: "a-again"}, {ID: 1}})
	if len(groups) != 1 || groups[0].Song.ID != 1 || groups[0].Song.Name != "a" || groups[0].Count != 3 {
		t.Fatalf("groups = %#v", groups)
	}
	actions := duplicateActions(groups, "pending")
	if len(actions) != 2 || actions[0].Op != "remove" || actions[1].Op != "add" {
		t.Fatalf("actions = %#v", actions)
	}
}

func TestParseSongIDFlagValuesDedupesInOrder(t *testing.T) {
	ids, err := parseSongIDFlagValues([]string{"3,1", "3", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 || ids[0] != 3 || ids[1] != 1 || ids[2] != 2 {
		t.Fatalf("ids = %#v", ids)
	}
	if _, err := parseSongIDFlagValues(nil); err == nil {
		t.Fatal("expected missing song-id error")
	}
}

func TestBuildTidyApplyPlan(t *testing.T) {
	sourcePlaylist := ncm.Playlist{ID: 10, Name: "source"}
	targetPlaylist := ncm.Playlist{ID: 20, Name: "target"}
	sourceSongs := []tidySong{{ID: 1, Name: "one"}, {ID: 2, Name: "two"}, {ID: 3, Name: "three"}}
	targetSongs := []tidySong{{ID: 2, Name: "two"}}

	result := buildTidyApplyPlan(tidyApplyOptions{
		SourceID: 10,
		TargetID: 20,
		SongIDs:  []int64{1, 2, 4, 1},
		Move:     true,
	}, sourcePlaylist, sourceSongs, targetPlaylist, targetSongs)

	if result.Summary.Matched != 2 || result.Summary.AlreadyInTarget != 1 || result.Summary.Pending != 2 || result.Summary.Skipped != 1 {
		t.Fatalf("summary = %#v", result.Summary)
	}
	addIDs := pendingActionSongIDs(result.Actions, "add")
	removeIDs := pendingActionSongIDs(result.Actions, "remove")
	if len(addIDs) != 1 || addIDs[0] != 1 {
		t.Fatalf("add ids = %#v", addIDs)
	}
	if len(removeIDs) != 2 || removeIDs[0] != 1 || removeIDs[1] != 2 {
		t.Fatalf("remove ids = %#v", removeIDs)
	}
}

func TestPlaylistTidyCommandValidation(t *testing.T) {
	cmd := newPlaylistTidyFilterCmd(&rootOptions{})
	cmd.SetArgs([]string{"123", "--playable", "maybe"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--playable 只支持") {
		t.Fatalf("err = %v", err)
	}

	cmd = newPlaylistTidyApplyCmd(&rootOptions{}, false)
	cmd.SetArgs([]string{"123", "--song-id", "1"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--to 必须指定") {
		t.Fatalf("err = %v", err)
	}

	cmd = newPlaylistTidyApplyCmd(&rootOptions{}, true)
	cmd.SetArgs([]string{"123", "--to", "456"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "至少需要一个 --song-id") {
		t.Fatalf("err = %v", err)
	}
}
