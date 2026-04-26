package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"ncm-cli/internal/ncm"
	"ncm-cli/internal/output"
)

const tidyPlaylistLimit = 10000

type tidyPlaylistRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type tidySong struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Artists    string `json:"artists"`
	Album      string `json:"album"`
	DurationMs int64  `json:"durationMs"`
	Playable   bool   `json:"playable"`
	Reason     string `json:"reason,omitempty"`
}

type tidySummary struct {
	Matched         int `json:"matched,omitempty"`
	AlreadyInTarget int `json:"alreadyInTarget,omitempty"`
	Pending         int `json:"pending,omitempty"`
	Intersection    int `json:"intersection,omitempty"`
	OnlySource      int `json:"onlySource,omitempty"`
	OnlyTarget      int `json:"onlyTarget,omitempty"`
	Duplicates      int `json:"duplicates,omitempty"`
	Added           int `json:"added,omitempty"`
	Removed         int `json:"removed,omitempty"`
	Skipped         int `json:"skipped,omitempty"`
}

type tidyResult struct {
	Source  *tidyPlaylistRef `json:"source,omitempty"`
	Target  *tidyPlaylistRef `json:"target,omitempty"`
	Summary tidySummary      `json:"summary"`
	Songs   []tidySong       `json:"songs,omitempty"`
	Actions []tidyAction     `json:"actions,omitempty"`
}

type tidyDiffResult struct {
	Source       tidyPlaylistRef `json:"source"`
	Target       tidyPlaylistRef `json:"target"`
	Summary      tidySummary     `json:"summary"`
	Intersection []tidySong      `json:"intersection"`
	OnlySource   []tidySong      `json:"onlySource"`
	OnlyTarget   []tidySong      `json:"onlyTarget"`
}

type tidyAction struct {
	Op      string `json:"op"`
	SongID  int64  `json:"songId"`
	Name    string `json:"name,omitempty"`
	Artists string `json:"artists,omitempty"`
	Album   string `json:"album,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Status  string `json:"status"`
}

type tidyFilter struct {
	Artist   string
	Album    string
	Name     string
	Playable string
}

type tidyDuplicateGroup struct {
	Song  tidySong `json:"song"`
	Count int      `json:"count"`
}

type tidyApplyOptions struct {
	SourceID int64
	TargetID int64
	SongIDs  []int64
	Move     bool
}

func newPlaylistTidyCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tidy",
		Short: "歌单整理辅助命令",
	}
	cmd.AddCommand(
		newPlaylistTidyInspectCmd(opts),
		newPlaylistTidyFilterCmd(opts),
		newPlaylistTidyDiffCmd(opts),
		newPlaylistTidyApplyCmd(opts, false),
		newPlaylistTidyApplyCmd(opts, true),
		newPlaylistTidyDuplicatesCmd(opts),
	)
	return cmd
}

func newPlaylistTidyInspectCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "inspect <playlist-id>",
		Short: "输出适合 Agent 分析的歌单歌曲清单",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			playlistID, err := parseRequiredID("playlist-id", args[0])
			if err != nil {
				return err
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			res, err := client.PlaylistDetail(ctx, playlistID, tidyPlaylistLimit)
			if err != nil {
				return err
			}
			songs := tidySongsFromDetail(res)
			result := tidyResult{
				Source:  tidyRef(res.Playlist),
				Summary: tidySummary{Matched: len(songs)},
				Songs:   songs,
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			if err := output.Text(cmd.OutOrStdout(), "%s (%d)\n", res.Playlist.Name, res.Playlist.TrackCount); err != nil {
				return err
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "ARTISTS", "ALBUM", "DURATION", "PLAY"}, tidySongRows(songs, false))
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistTidyFilterCmd(opts *rootOptions) *cobra.Command {
	var filter tidyFilter
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "filter <playlist-id>",
		Short: "按基础条件筛选歌单歌曲",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateTidyFilter(filter); err != nil {
				return err
			}
			playlistID, err := parseRequiredID("playlist-id", args[0])
			if err != nil {
				return err
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			res, err := client.PlaylistDetail(ctx, playlistID, tidyPlaylistLimit)
			if err != nil {
				return err
			}
			matched := filterTidySongs(tidySongsFromDetail(res), filter)
			result := tidyResult{
				Source:  tidyRef(res.Playlist),
				Summary: tidySummary{Matched: len(matched)},
				Songs:   matched,
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "ARTISTS", "ALBUM", "DURATION", "PLAY", "REASON"}, tidySongRows(matched, true))
		},
	}
	cmd.Flags().StringVar(&filter.Artist, "artist", "", "按歌手名包含筛选")
	cmd.Flags().StringVar(&filter.Album, "album", "", "按专辑名包含筛选")
	cmd.Flags().StringVar(&filter.Name, "name", "", "按歌曲名包含筛选")
	cmd.Flags().StringVar(&filter.Playable, "playable", "", "按可播放状态筛选: yes 或 no")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistTidyDiffCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "diff <source-playlist-id> <target-playlist-id>",
		Short: "比较两个歌单的歌曲差异",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceID, err := parseRequiredID("source-playlist-id", args[0])
			if err != nil {
				return err
			}
			targetID, err := parseRequiredID("target-playlist-id", args[1])
			if err != nil {
				return err
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			source, target, err := fetchTwoPlaylists(ctx, client, sourceID, targetID)
			if err != nil {
				return err
			}
			result := diffTidySongs(source.Playlist, tidySongsFromDetail(source), target.Playlist, tidySongsFromDetail(target))
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			return output.Table(cmd.OutOrStdout(), []string{"SECTION", "ID", "NAME", "ARTISTS", "ALBUM", "PLAY"}, tidyDiffRows(result))
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistTidyApplyCmd(opts *rootOptions, move bool) *cobra.Command {
	var targetID int64
	var songIDValues []string
	var yes bool
	var asJSON bool
	use := "apply <source-playlist-id>"
	short := "把指定歌曲复制到目标歌单"
	if move {
		use = "move <source-playlist-id>"
		short = "把指定歌曲移动到目标歌单"
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceID, err := parseRequiredID("source-playlist-id", args[0])
			if err != nil {
				return err
			}
			if targetID <= 0 {
				return fmt.Errorf("--to 必须指定有效的目标歌单 ID")
			}
			songIDs, err := parseSongIDFlagValues(songIDValues)
			if err != nil {
				return err
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			if move {
				if _, err := requireOwnedPlaylist(ctx, client, sourceID); err != nil {
					return err
				}
			}
			targetPlaylist, err := requireOwnedPlaylist(ctx, client, targetID)
			if err != nil {
				return err
			}
			source, target, err := fetchTwoPlaylists(ctx, client, sourceID, targetID)
			if err != nil {
				return err
			}
			result := buildTidyApplyPlan(tidyApplyOptions{
				SourceID: sourceID,
				TargetID: targetID,
				SongIDs:  songIDs,
				Move:     move,
			}, source.Playlist, tidySongsFromDetail(source), target.Playlist, tidySongsFromDetail(target))
			addIDs := pendingActionSongIDs(result.Actions, "add")
			removeIDs := pendingActionSongIDs(result.Actions, "remove")
			writeCount := len(addIDs)
			if move {
				writeCount = len(removeIDs)
			}
			if writeCount > 0 {
				prompt := fmt.Sprintf("确认将 %d 首歌曲复制到歌单 %s？输入 yes 继续: ", len(addIDs), playlistLabel(targetPlaylist))
				if move {
					prompt = fmt.Sprintf("确认将 %d 首歌曲移动到歌单 %s？输入 yes 继续: ", len(removeIDs), playlistLabel(targetPlaylist))
				}
				if err := confirmExpected(cmd, yes, prompt, "yes"); err != nil {
					return err
				}
				if len(addIDs) > 0 {
					res, err := client.PlaylistAddTracks(ctx, targetID, addIDs)
					if err != nil {
						return err
					}
					markActions(result.Actions, "add", "added")
					result.Summary.Added = len(addIDs)
					if res != nil && res.Code != 0 && res.Code != 200 {
						return fmt.Errorf("添加歌曲失败: %s", writeMessage(res))
					}
				}
				if move {
					res, err := client.PlaylistRemoveTracks(ctx, sourceID, removeIDs)
					if err != nil {
						markActions(result.Actions, "remove", "failed")
						if asJSON {
							_ = output.JSON(cmd.OutOrStdout(), result)
						}
						return err
					}
					markActions(result.Actions, "remove", "removed")
					result.Summary.Removed = len(removeIDs)
					if res != nil && res.Code != 0 && res.Code != 200 {
						return fmt.Errorf("移除源歌单歌曲失败: %s", writeMessage(res))
					}
				}
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			if move {
				return output.Text(cmd.OutOrStdout(), "已移动 %d 首歌曲到歌单 %s。\n", result.Summary.Removed, playlistLabel(targetPlaylist))
			}
			return output.Text(cmd.OutOrStdout(), "已复制 %d 首歌曲到歌单 %s。\n", result.Summary.Added, playlistLabel(targetPlaylist))
		},
	}
	cmd.Flags().Int64Var(&targetID, "to", 0, "目标歌单 ID")
	cmd.Flags().StringArrayVar(&songIDValues, "song-id", nil, "歌曲 ID，可重复传入")
	cmd.Flags().BoolVar(&yes, "yes", false, "跳过确认")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistTidyDuplicatesCmd(opts *rootOptions) *cobra.Command {
	var apply bool
	var yes bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "duplicates <playlist-id>",
		Short: "扫描同一歌单内重复歌曲",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			playlistID, err := parseRequiredID("playlist-id", args[0])
			if err != nil {
				return err
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			if apply {
				if _, err := requireOwnedPlaylist(ctx, client, playlistID); err != nil {
					return err
				}
			}
			res, err := client.PlaylistDetail(ctx, playlistID, tidyPlaylistLimit)
			if err != nil {
				return err
			}
			groups := duplicateTidySongs(tidySongsFromDetail(res))
			result := tidyResult{
				Source:  tidyRef(res.Playlist),
				Summary: tidySummary{Duplicates: len(groups)},
				Actions: duplicateActions(groups, "pending"),
			}
			if apply && len(groups) > 0 {
				ids := duplicateGroupIDs(groups)
				prompt := fmt.Sprintf("确认清理歌单 %s 中 %d 首重复歌曲？输入 yes 继续: ", playlistLabel(&res.Playlist), len(ids))
				if err := confirmExpected(cmd, yes, prompt, "yes"); err != nil {
					return err
				}
				if _, err := client.PlaylistRemoveTracks(ctx, playlistID, ids); err != nil {
					markActions(result.Actions, "remove", "failed")
					if asJSON {
						_ = output.JSON(cmd.OutOrStdout(), result)
					}
					return err
				}
				markActions(result.Actions, "remove", "removed")
				result.Summary.Removed = len(ids)
				if _, err := client.PlaylistAddTracks(ctx, playlistID, ids); err != nil {
					markActions(result.Actions, "add", "failed")
					if asJSON {
						_ = output.JSON(cmd.OutOrStdout(), result)
					}
					return err
				}
				markActions(result.Actions, "add", "added")
				result.Summary.Added = len(ids)
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			if len(groups) == 0 {
				return output.Text(cmd.OutOrStdout(), "未发现重复歌曲。\n")
			}
			if apply {
				return output.Text(cmd.OutOrStdout(), "已清理 %d 首重复歌曲；重复歌曲会被重新加入一次，位置可能变化。\n", len(groups))
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "ARTISTS", "ALBUM", "COUNT", "ACTION"}, duplicateRows(groups))
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "清理重复歌曲")
	cmd.Flags().BoolVar(&yes, "yes", false, "跳过确认")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func fetchTwoPlaylists(ctx context.Context, client *ncm.Client, sourceID, targetID int64) (*ncm.PlaylistDetailResponse, *ncm.PlaylistDetailResponse, error) {
	source, err := client.PlaylistDetail(ctx, sourceID, tidyPlaylistLimit)
	if err != nil {
		return nil, nil, err
	}
	target, err := client.PlaylistDetail(ctx, targetID, tidyPlaylistLimit)
	if err != nil {
		return nil, nil, err
	}
	return source, target, nil
}

func tidyRef(playlist ncm.Playlist) *tidyPlaylistRef {
	return &tidyPlaylistRef{ID: playlist.ID, Name: playlist.Name}
}

func tidySongsFromDetail(res *ncm.PlaylistDetailResponse) []tidySong {
	if res == nil {
		return nil
	}
	privileges := privilegeByID(res.Privileges)
	songs := make([]tidySong, 0, len(res.Playlist.Tracks))
	for _, song := range res.Playlist.Tracks {
		priv := privileges[song.ID]
		songs = append(songs, tidySong{
			ID:         song.ID,
			Name:       song.Name,
			Artists:    ncm.ArtistsText(song.Artists),
			Album:      song.Album.Name,
			DurationMs: song.Duration,
			Playable:   priv.ID != 0 && priv.PL > 0,
		})
	}
	return songs
}

func validateTidyFilter(filter tidyFilter) error {
	if filter.Artist == "" && filter.Album == "" && filter.Name == "" && filter.Playable == "" {
		return fmt.Errorf("至少需要指定一个筛选条件")
	}
	if filter.Playable != "" && filter.Playable != "yes" && filter.Playable != "no" {
		return fmt.Errorf("--playable 只支持 yes 或 no")
	}
	return nil
}

func filterTidySongs(songs []tidySong, filter tidyFilter) []tidySong {
	out := make([]tidySong, 0, len(songs))
	for _, song := range songs {
		reasons := tidyFilterReasons(song, filter)
		if reasons == nil {
			continue
		}
		song.Reason = strings.Join(reasons, "; ")
		out = append(out, song)
	}
	return out
}

func tidyFilterReasons(song tidySong, filter tidyFilter) []string {
	reasons := make([]string, 0, 4)
	if filter.Artist != "" {
		if !containsFold(song.Artists, filter.Artist) {
			return nil
		}
		reasons = append(reasons, "artist contains "+filter.Artist)
	}
	if filter.Album != "" {
		if !containsFold(song.Album, filter.Album) {
			return nil
		}
		reasons = append(reasons, "album contains "+filter.Album)
	}
	if filter.Name != "" {
		if !containsFold(song.Name, filter.Name) {
			return nil
		}
		reasons = append(reasons, "name contains "+filter.Name)
	}
	if filter.Playable != "" {
		wantPlayable := filter.Playable == "yes"
		if song.Playable != wantPlayable {
			return nil
		}
		reasons = append(reasons, "playable is "+filter.Playable)
	}
	return reasons
}

func diffTidySongs(sourcePlaylist ncm.Playlist, sourceSongs []tidySong, targetPlaylist ncm.Playlist, targetSongs []tidySong) tidyDiffResult {
	targetByID := tidySongSet(targetSongs)
	sourceByID := tidySongSet(sourceSongs)
	result := tidyDiffResult{
		Source: tidyPlaylistRef{ID: sourcePlaylist.ID, Name: sourcePlaylist.Name},
		Target: tidyPlaylistRef{ID: targetPlaylist.ID, Name: targetPlaylist.Name},
	}
	for _, song := range sourceSongs {
		if _, ok := targetByID[song.ID]; ok {
			result.Intersection = append(result.Intersection, song)
			continue
		}
		result.OnlySource = append(result.OnlySource, song)
	}
	for _, song := range targetSongs {
		if _, ok := sourceByID[song.ID]; !ok {
			result.OnlyTarget = append(result.OnlyTarget, song)
		}
	}
	result.Summary.Intersection = len(result.Intersection)
	result.Summary.OnlySource = len(result.OnlySource)
	result.Summary.OnlyTarget = len(result.OnlyTarget)
	return result
}

func buildTidyApplyPlan(options tidyApplyOptions, sourcePlaylist ncm.Playlist, sourceSongs []tidySong, targetPlaylist ncm.Playlist, targetSongs []tidySong) tidyResult {
	sourceByID := tidySongSet(sourceSongs)
	targetByID := tidySongSet(targetSongs)
	result := tidyResult{
		Source: &tidyPlaylistRef{ID: sourcePlaylist.ID, Name: sourcePlaylist.Name},
		Target: &tidyPlaylistRef{ID: targetPlaylist.ID, Name: targetPlaylist.Name},
	}
	for _, id := range uniqueInt64s(options.SongIDs) {
		song, ok := sourceByID[id]
		action := tidyAction{Op: "add", SongID: id, Status: "skipped"}
		if ok {
			action.Name = song.Name
			action.Artists = song.Artists
			action.Album = song.Album
		}
		if !ok {
			action.Reason = "not in source playlist"
			result.Summary.Skipped++
			result.Actions = append(result.Actions, action)
			continue
		}
		result.Summary.Matched++
		if _, exists := targetByID[id]; exists {
			action.Reason = "already in target"
			action.Status = "already_in_target"
			result.Summary.AlreadyInTarget++
			if options.Move {
				result.Summary.Pending++
				result.Actions = append(result.Actions, tidyAction{
					Op:      "remove",
					SongID:  id,
					Name:    song.Name,
					Artists: song.Artists,
					Album:   song.Album,
					Reason:  "target already has song",
					Status:  "pending",
				})
			}
			result.Actions = append(result.Actions, action)
			continue
		}
		action.Reason = "selected by song-id"
		action.Status = "pending"
		result.Summary.Pending++
		result.Actions = append(result.Actions, action)
		if options.Move {
			result.Actions = append(result.Actions, tidyAction{
				Op:      "remove",
				SongID:  id,
				Name:    song.Name,
				Artists: song.Artists,
				Album:   song.Album,
				Reason:  "move after add",
				Status:  "pending",
			})
		}
	}
	return result
}

func duplicateTidySongs(songs []tidySong) []tidyDuplicateGroup {
	first := make(map[int64]tidySong, len(songs))
	counts := make(map[int64]int, len(songs))
	order := make([]int64, 0)
	for _, song := range songs {
		if _, ok := first[song.ID]; !ok {
			first[song.ID] = song
			order = append(order, song.ID)
		}
		counts[song.ID]++
	}
	groups := make([]tidyDuplicateGroup, 0)
	for _, id := range order {
		if counts[id] > 1 {
			groups = append(groups, tidyDuplicateGroup{Song: first[id], Count: counts[id]})
		}
	}
	return groups
}

func duplicateActions(groups []tidyDuplicateGroup, status string) []tidyAction {
	actions := make([]tidyAction, 0, len(groups)*2)
	for _, group := range groups {
		reason := fmt.Sprintf("duplicate count %d", group.Count)
		actions = append(actions, tidyAction{
			Op:      "remove",
			SongID:  group.Song.ID,
			Name:    group.Song.Name,
			Artists: group.Song.Artists,
			Album:   group.Song.Album,
			Reason:  reason,
			Status:  status,
		})
		actions = append(actions, tidyAction{
			Op:      "add",
			SongID:  group.Song.ID,
			Name:    group.Song.Name,
			Artists: group.Song.Artists,
			Album:   group.Song.Album,
			Reason:  "re-add one copy after cleanup",
			Status:  status,
		})
	}
	return actions
}

func duplicateGroupIDs(groups []tidyDuplicateGroup) []int64 {
	ids := make([]int64, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.Song.ID)
	}
	return ids
}

func tidySongSet(songs []tidySong) map[int64]tidySong {
	out := make(map[int64]tidySong, len(songs))
	for _, song := range songs {
		if _, exists := out[song.ID]; !exists {
			out[song.ID] = song
		}
	}
	return out
}

func parseSongIDFlagValues(values []string) ([]int64, error) {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				parts = append(parts, part)
			}
		}
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("至少需要一个 --song-id")
	}
	ids, err := parseSongIDs(parts)
	if err != nil {
		return nil, err
	}
	return uniqueInt64s(ids), nil
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]bool, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func pendingActionSongIDs(actions []tidyAction, op string) []int64 {
	ids := make([]int64, 0, len(actions))
	for _, action := range actions {
		if action.Op == op && action.Status == "pending" {
			ids = append(ids, action.SongID)
		}
	}
	return ids
}

func markActions(actions []tidyAction, op string, status string) {
	for i := range actions {
		if actions[i].Op == op && actions[i].Status == "pending" {
			actions[i].Status = status
		}
	}
}

func containsFold(text, needle string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(needle))
}

func tidySongRows(songs []tidySong, withReason bool) [][]string {
	rows := make([][]string, 0, len(songs))
	for _, song := range songs {
		row := []string{
			strconv.FormatInt(song.ID, 10),
			song.Name,
			song.Artists,
			song.Album,
			formatDuration(song.DurationMs),
			playableText(song.Playable),
		}
		if withReason {
			row = append(row, song.Reason)
		}
		rows = append(rows, row)
	}
	return rows
}

func tidyDiffRows(result tidyDiffResult) [][]string {
	rows := make([][]string, 0, len(result.Intersection)+len(result.OnlySource)+len(result.OnlyTarget))
	appendRows := func(section string, songs []tidySong) {
		for _, song := range songs {
			rows = append(rows, []string{
				section,
				strconv.FormatInt(song.ID, 10),
				song.Name,
				song.Artists,
				song.Album,
				playableText(song.Playable),
			})
		}
	}
	appendRows("both", result.Intersection)
	appendRows("only_source", result.OnlySource)
	appendRows("only_target", result.OnlyTarget)
	return rows
}

func duplicateRows(groups []tidyDuplicateGroup) [][]string {
	rows := make([][]string, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, []string{
			strconv.FormatInt(group.Song.ID, 10),
			group.Song.Name,
			group.Song.Artists,
			group.Song.Album,
			strconv.Itoa(group.Count),
			"remove duplicates",
		})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left, _ := strconv.ParseInt(rows[i][0], 10, 64)
		right, _ := strconv.ParseInt(rows[j][0], 10, 64)
		return left < right
	})
	return rows
}

func playableText(playable bool) string {
	if playable {
		return "yes"
	}
	return "no"
}
