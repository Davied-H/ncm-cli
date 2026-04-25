package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"ncm-cli/internal/checktoken"
	"ncm-cli/internal/ncm"
	"ncm-cli/internal/output"
)

type playlistWriteResult struct {
	Code        int           `json:"code"`
	Message     string        `json:"message,omitempty"`
	Playlist    *ncm.Playlist `json:"playlist,omitempty"`
	PlaylistID  int64         `json:"playlistId,omitempty"`
	SongIDs     []int64       `json:"songIds,omitempty"`
	Name        string        `json:"name,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Description string        `json:"description,omitempty"`
}

func newPlaylistCreateCmd(opts *rootOptions) *cobra.Command {
	var private bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "创建歌单",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("歌单名不能为空")
			}
			client, statePath, profileDir, err := makeClientWithSession(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			token, err := checktoken.Get(ctx, profileDir, statePath, opts.timeout)
			if err != nil {
				return err
			}
			res, err := client.PlaylistCreate(ctx, name, private, token)
			if err != nil {
				return err
			}
			result := newPlaylistWriteResult(res)
			result.Playlist = res.Playlist
			if res.Playlist != nil {
				result.PlaylistID = res.Playlist.ID
				result.Name = res.Playlist.Name
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			if res.Playlist == nil {
				return output.Text(cmd.OutOrStdout(), "创建歌单完成：%s\n", writeMessage(res))
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "TRACKS", "PRIVACY"}, [][]string{{
				strconv.FormatInt(res.Playlist.ID, 10),
				res.Playlist.Name,
				strconv.Itoa(res.Playlist.TrackCount),
				privacyText(res.Playlist, private),
			}})
		},
	}
	cmd.Flags().BoolVar(&private, "private", false, "创建私密歌单")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistAddCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "add <playlist-id> <song-id...>",
		Short: "添加歌曲到歌单",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			playlistID, err := parseRequiredID("playlist-id", args[0])
			if err != nil {
				return err
			}
			songIDs, err := parseSongIDs(args[1:])
			if err != nil {
				return err
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			playlist, err := requireOwnedPlaylist(ctx, client, playlistID)
			if err != nil {
				return err
			}
			res, err := client.PlaylistAddTracks(ctx, playlistID, songIDs)
			if err != nil {
				return err
			}
			result := newPlaylistWriteResult(res)
			result.PlaylistID = playlistID
			result.SongIDs = songIDs
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			return output.Text(cmd.OutOrStdout(), "已添加 %d 首歌曲到歌单 %s。\n", len(songIDs), playlistLabel(playlist))
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistRemoveCmd(opts *rootOptions) *cobra.Command {
	var yes bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "remove <playlist-id> <song-id...>",
		Short: "从歌单移除歌曲",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			playlistID, err := parseRequiredID("playlist-id", args[0])
			if err != nil {
				return err
			}
			songIDs, err := parseSongIDs(args[1:])
			if err != nil {
				return err
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			playlist, err := requireOwnedPlaylist(ctx, client, playlistID)
			if err != nil {
				return err
			}
			if err := confirmExpected(cmd, yes, fmt.Sprintf("确认从歌单 %s 移除 %d 首歌曲？输入 yes 继续: ", playlistLabel(playlist), len(songIDs)), "yes"); err != nil {
				return err
			}
			res, err := client.PlaylistRemoveTracks(ctx, playlistID, songIDs)
			if err != nil {
				return err
			}
			result := newPlaylistWriteResult(res)
			result.PlaylistID = playlistID
			result.SongIDs = songIDs
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			return output.Text(cmd.OutOrStdout(), "已从歌单 %s 移除 %d 首歌曲。\n", playlistLabel(playlist), len(songIDs))
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "跳过确认")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistDeleteCmd(opts *rootOptions) *cobra.Command {
	var yes bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "delete <playlist-id>",
		Short: "删除歌单",
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
			playlist, err := requireOwnedPlaylist(ctx, client, playlistID)
			if err != nil {
				return err
			}
			expected := strconv.FormatInt(playlistID, 10)
			if err := confirmExpected(cmd, yes, fmt.Sprintf("确认删除歌单 %s？输入歌单 ID 继续: ", playlistLabel(playlist)), expected); err != nil {
				return err
			}
			res, err := client.PlaylistDelete(ctx, playlistID)
			if err != nil {
				return err
			}
			result := newPlaylistWriteResult(res)
			result.PlaylistID = playlistID
			result.Name = playlist.Name
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			return output.Text(cmd.OutOrStdout(), "已删除歌单 %s。\n", playlistLabel(playlist))
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "跳过确认")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistRenameCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "rename <playlist-id> <name>",
		Short: "重命名歌单",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			playlistID, err := parseRequiredID("playlist-id", args[0])
			if err != nil {
				return err
			}
			name := strings.TrimSpace(args[1])
			if name == "" {
				return fmt.Errorf("歌单名不能为空")
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			playlist, err := requireOwnedPlaylist(ctx, client, playlistID)
			if err != nil {
				return err
			}
			res, err := client.PlaylistRename(ctx, playlistID, name)
			if err != nil {
				return err
			}
			result := newPlaylistWriteResult(res)
			result.PlaylistID = playlistID
			result.Name = name
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			return output.Text(cmd.OutOrStdout(), "已将歌单 %s 重命名为 %s。\n", playlistLabel(playlist), name)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistTagsCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "tags <playlist-id> <tag...>",
		Short: "更新歌单标签",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			playlistID, err := parseRequiredID("playlist-id", args[0])
			if err != nil {
				return err
			}
			tags := cleanTags(args[1:])
			if len(tags) == 0 {
				return fmt.Errorf("至少需要一个非空标签")
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			playlist, err := requireOwnedPlaylist(ctx, client, playlistID)
			if err != nil {
				return err
			}
			res, err := client.PlaylistUpdateTags(ctx, playlistID, tags)
			if err != nil {
				return err
			}
			result := newPlaylistWriteResult(res)
			result.PlaylistID = playlistID
			result.Tags = tags
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			return output.Text(cmd.OutOrStdout(), "已更新歌单 %s 标签：%s。\n", playlistLabel(playlist), strings.Join(tags, ","))
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistDescCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "desc <playlist-id> <text>",
		Short: "更新歌单描述",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			playlistID, err := parseRequiredID("playlist-id", args[0])
			if err != nil {
				return err
			}
			desc := args[1]
			client, statePath, profileDir, err := makeClientWithSession(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			playlist, err := requireOwnedPlaylist(ctx, client, playlistID)
			if err != nil {
				return err
			}
			token, err := checktoken.Get(ctx, profileDir, statePath, opts.timeout)
			if err != nil {
				return err
			}
			res, err := client.PlaylistUpdateDescription(ctx, playlistID, desc, token)
			if err != nil {
				return err
			}
			result := newPlaylistWriteResult(res)
			result.PlaylistID = playlistID
			result.Description = desc
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), result)
			}
			return output.Text(cmd.OutOrStdout(), "已更新歌单 %s 描述。\n", playlistLabel(playlist))
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func requireOwnedPlaylist(ctx context.Context, client *ncm.Client, playlistID int64) (*ncm.Playlist, error) {
	me, err := client.Me(ctx)
	if err != nil {
		return nil, err
	}
	if me.Profile == nil || me.Profile.UserID == 0 {
		return nil, fmt.Errorf("无法从账号响应获取 userId")
	}
	res, err := client.PlaylistDetail(ctx, playlistID, 1)
	if err != nil {
		return nil, err
	}
	playlist := &res.Playlist
	if playlist.SpecialType == 5 {
		return nil, fmt.Errorf("不支持操作“我喜欢的音乐”等特殊歌单")
	}
	ownerID := playlist.UserID
	if ownerID == 0 && playlist.Creator != nil {
		ownerID = playlist.Creator.UserID
	}
	if ownerID == 0 {
		return nil, fmt.Errorf("无法确认歌单归属，已取消写操作")
	}
	if ownerID != me.Profile.UserID {
		return nil, fmt.Errorf("只能操作当前账号自建歌单，目标歌单 owner=%d，当前账号=%d", ownerID, me.Profile.UserID)
	}
	return playlist, nil
}

func parseRequiredID(name, value string) (int64, error) {
	id, err := ncm.ParseID(value)
	if err != nil || id <= 0 {
		if err == nil {
			err = fmt.Errorf("必须大于 0")
		}
		return 0, fmt.Errorf("%s 无效: %w", name, err)
	}
	return id, nil
}

func parseSongIDs(values []string) ([]int64, error) {
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		id, err := parseRequiredID("song-id", value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("至少需要一个 song-id")
	}
	return ids, nil
}

func cleanTags(values []string) []string {
	tags := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			tag := strings.TrimSpace(part)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

func confirmExpected(cmd *cobra.Command, yes bool, prompt string, expected string) error {
	if yes {
		return nil
	}
	if _, err := fmt.Fprint(cmd.ErrOrStderr(), prompt); err != nil {
		return err
	}
	got, err := readPromptLine(cmd.InOrStdin())
	if err != nil {
		return err
	}
	if got != expected {
		return fmt.Errorf("未确认，已取消")
	}
	return nil
}

func readPromptLine(r io.Reader) (string, error) {
	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func newPlaylistWriteResult(res *ncm.PlaylistWriteResponse) playlistWriteResult {
	result := playlistWriteResult{}
	if res == nil {
		return result
	}
	result.Code = res.Code
	result.Message = res.MessageText()
	return result
}

func writeMessage(res *ncm.PlaylistWriteResponse) string {
	msg := res.MessageText()
	if msg == "" {
		return "ok"
	}
	return msg
}

func playlistLabel(playlist *ncm.Playlist) string {
	if playlist == nil {
		return ""
	}
	if playlist.Name == "" {
		return fmt.Sprintf("(%d)", playlist.ID)
	}
	return fmt.Sprintf("%s (%d)", playlist.Name, playlist.ID)
}

func privacyText(playlist *ncm.Playlist, fallbackPrivate bool) string {
	if playlist != nil {
		if playlist.Privacy == 10 {
			return "private"
		}
		if playlist.Privacy == 0 {
			return "public"
		}
		return strconv.Itoa(playlist.Privacy)
	}
	if fallbackPrivate {
		return "private"
	}
	return "public"
}
