package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"ncm-cli/internal/config"
	"ncm-cli/internal/desktop"
	"ncm-cli/internal/login"
	"ncm-cli/internal/ncm"
	"ncm-cli/internal/output"
)

var (
	version = "dev"
	commit  = "unknown"
)

type rootOptions struct {
	configDir string
	timeout   time.Duration
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{timeout: 30 * time.Second}
	cmd := &cobra.Command{
		Use:           "ncm",
		Short:         "网易云音乐命令行工具",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.PersistentFlags().StringVar(&opts.configDir, "config-dir", "", "配置目录，默认使用 NCM_CONFIG_DIR 或 ~/.config/ncm-cli")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", opts.timeout, "请求超时时间")
	cmd.AddCommand(
		newLoginCmd(opts),
		newMeCmd(opts),
		newPlaylistCmd(opts),
		newSongCmd(opts),
		newPlayCmd(),
		newLyricCmd(opts),
		newSearchCmd(opts),
		newVersionCmd(),
	)
	return cmd
}

func newVersionCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "显示 ncm 版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := struct {
				Version string `json:"version"`
				Commit  string `json:"commit"`
			}{
				Version: version,
				Commit:  commit,
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), info)
			}
			return output.Text(cmd.OutOrStdout(), "ncm %s (%s)\n", version, commit)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newLoginCmd(opts *rootOptions) *cobra.Command {
	var headless bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "打开浏览器登录网易云音乐",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
			defer cancel()
			info, err := login.Run(ctx, login.Options{
				ConfigDir: opts.configDir,
				Headless:  headless,
				Timeout:   10 * time.Minute,
				Stdout:    cmd.OutOrStdout(),
			})
			if err != nil {
				return err
			}
			return output.JSON(cmd.OutOrStdout(), info)
		},
	}
	cmd.Flags().BoolVar(&headless, "headless", false, "使用无头浏览器登录")
	return cmd
}

func newMeCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "me",
		Short: "显示当前账号",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			me, err := client.Me(ctx)
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), me)
			}
			if me.Profile == nil {
				return fmt.Errorf("账号响应缺少 profile")
			}
			return output.Table(cmd.OutOrStdout(), []string{"USER_ID", "NICKNAME"}, [][]string{{
				strconv.FormatInt(me.Profile.UserID, 10),
				me.Profile.Nickname,
			}})
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "playlist",
		Short: "歌单相关命令",
	}
	cmd.AddCommand(
		newPlaylistListCmd(opts),
		newPlaylistShowCmd(opts),
		newPlaylistCreateCmd(opts),
		newPlaylistAddCmd(opts),
		newPlaylistRemoveCmd(opts),
		newPlaylistDeleteCmd(opts),
		newPlaylistRenameCmd(opts),
		newPlaylistTagsCmd(opts),
		newPlaylistDescCmd(opts),
	)
	return cmd
}

func newPlaylistListCmd(opts *rootOptions) *cobra.Command {
	var uid int64
	var limit int
	var offset int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出用户歌单",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit 必须大于 0")
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			effectiveUID := uid
			if effectiveUID == 0 {
				me, err := client.Me(ctx)
				if err != nil {
					return err
				}
				if me.Profile == nil || me.Profile.UserID == 0 {
					return fmt.Errorf("无法从账号响应获取 userId")
				}
				effectiveUID = me.Profile.UserID
			}
			res, err := client.PlaylistList(ctx, effectiveUID, limit, offset)
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			rows := make([][]string, 0, len(res.Playlist))
			for _, playlist := range res.Playlist {
				kind := "收藏"
				if playlist.UserID == effectiveUID {
					kind = "自建"
				}
				rows = append(rows, []string{
					strconv.FormatInt(playlist.ID, 10),
					playlist.Name,
					strconv.Itoa(playlist.TrackCount),
					kind,
				})
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "TRACKS", "TYPE"}, rows)
		},
	}
	cmd.Flags().Int64Var(&uid, "uid", 0, "用户 ID，默认当前登录用户")
	cmd.Flags().IntVar(&limit, "limit", 100, "返回数量")
	cmd.Flags().IntVar(&offset, "offset", 0, "分页偏移")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlaylistShowCmd(opts *rootOptions) *cobra.Command {
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show <playlist-id>",
		Short: "显示歌单歌曲",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit 必须大于 0")
			}
			id, err := ncm.ParseID(args[0])
			if err != nil {
				return fmt.Errorf("playlist-id 无效: %w", err)
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			res, err := client.PlaylistDetail(ctx, id, limit)
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			rows := make([][]string, 0, len(res.Playlist.Tracks))
			privileges := privilegeByID(res.Privileges)
			for _, song := range res.Playlist.Tracks {
				priv := privileges[song.ID]
				rows = append(rows, []string{
					strconv.FormatInt(song.ID, 10),
					song.Name,
					ncm.ArtistsText(song.Artists),
					song.Album.Name,
					formatDuration(song.Duration),
					formatPrivilege(priv),
				})
			}
			if err := output.Text(cmd.OutOrStdout(), "%s (%d)\n", res.Playlist.Name, res.Playlist.TrackCount); err != nil {
				return err
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "ARTISTS", "ALBUM", "DURATION", "PLAY"}, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 1000, "返回歌曲数量")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newSongCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "song <song-id>",
		Short: "显示歌曲元数据",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := ncm.ParseID(args[0])
			if err != nil {
				return fmt.Errorf("song-id 无效: %w", err)
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			res, err := client.SongDetail(ctx, id)
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			privileges := privilegeByID(res.Privileges)
			rows := make([][]string, 0, len(res.Songs))
			for _, song := range res.Songs {
				rows = append(rows, []string{
					strconv.FormatInt(song.ID, 10),
					song.Name,
					ncm.ArtistsText(song.Artists),
					song.Album.Name,
					formatDuration(song.Duration),
					formatPrivilege(privileges[song.ID]),
				})
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "ARTISTS", "ALBUM", "DURATION", "PLAY"}, rows)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newPlayCmd() *cobra.Command {
	var printURL bool
	cmd := &cobra.Command{
		Use:   "play <song-id>",
		Short: "用网易云音乐桌面端播放歌曲",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := ncm.ParseID(args[0])
			if err != nil {
				return fmt.Errorf("song-id 无效: %w", err)
			}
			url, err := desktop.SongPlayURL(id)
			if err != nil {
				return err
			}
			if printURL {
				if err := output.Text(cmd.OutOrStdout(), "%s\n", url); err != nil {
					return err
				}
			}
			return desktop.Open(url)
		},
	}
	cmd.Flags().BoolVar(&printURL, "print-url", false, "同时输出将调用的 orpheus URL")
	return cmd
}

func newLyricCmd(opts *rootOptions) *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "lyric <song-id>",
		Short: "显示歌词",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := ncm.ParseID(args[0])
			if err != nil {
				return fmt.Errorf("song-id 无效: %w", err)
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			res, err := client.Lyric(ctx, id)
			if err != nil {
				return err
			}
			if raw {
				return output.Text(cmd.OutOrStdout(), "%s", res.LRC.Lyric)
			}
			return output.JSON(cmd.OutOrStdout(), res)
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "只输出原始歌词")
	return cmd
}

func newSearchCmd(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "搜索命令",
	}
	cmd.AddCommand(newSearchSuggestCmd(opts), newSearchSongCmd(opts), newSearchPlaylistCmd(opts))
	return cmd
}

func newSearchSuggestCmd(opts *rootOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "suggest <keyword>",
		Short: "搜索建议",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			res, err := client.SearchSuggest(ctx, args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			rows := make([][]string, 0, len(res.Result.Songs)+len(res.Result.Artists)+len(res.Result.Albums))
			for _, song := range res.Result.Songs {
				rows = append(rows, []string{"song", strconv.FormatInt(song.ID, 10), song.Name, ncm.ArtistsText(song.Artists)})
			}
			for _, artist := range res.Result.Artists {
				rows = append(rows, []string{"artist", strconv.FormatInt(artist.ID, 10), artist.Name, ""})
			}
			for _, album := range res.Result.Albums {
				rows = append(rows, []string{"album", strconv.FormatInt(album.ID, 10), album.Name, ""})
			}
			return output.Table(cmd.OutOrStdout(), []string{"TYPE", "ID", "NAME", "EXTRA"}, rows)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newSearchSongCmd(opts *rootOptions) *cobra.Command {
	var limit int
	var offset int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "song <keyword>",
		Short: "搜索歌曲",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit 必须大于 0")
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			res, err := client.SearchSongs(ctx, args[0], limit, offset)
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			rows := make([][]string, 0, len(res.Result.Songs))
			for _, song := range res.Result.Songs {
				rows = append(rows, []string{
					strconv.FormatInt(song.ID, 10),
					song.Name,
					ncm.ArtistsText(song.Artists),
					song.Album.Name,
					formatDuration(song.Duration),
				})
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "ARTISTS", "ALBUM", "DURATION"}, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 30, "返回数量")
	cmd.Flags().IntVar(&offset, "offset", 0, "分页偏移")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func newSearchPlaylistCmd(opts *rootOptions) *cobra.Command {
	var limit int
	var offset int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "playlist <keyword>",
		Short: "搜索歌单",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit 必须大于 0")
			}
			client, err := makeClient(opts)
			if err != nil {
				return err
			}
			ctx, cancel := commandContext(cmd, opts)
			defer cancel()
			res, err := client.SearchPlaylists(ctx, args[0], limit, offset)
			if err != nil {
				return err
			}
			if asJSON {
				return output.JSON(cmd.OutOrStdout(), res)
			}
			rows := make([][]string, 0, len(res.Result.Playlists))
			for _, playlist := range res.Result.Playlists {
				creator := ""
				if playlist.Creator != nil {
					creator = playlist.Creator.Nickname
				}
				rows = append(rows, []string{
					strconv.FormatInt(playlist.ID, 10),
					playlist.Name,
					strconv.Itoa(playlist.TrackCount),
					creator,
				})
			}
			return output.Table(cmd.OutOrStdout(), []string{"ID", "NAME", "TRACKS", "CREATOR"}, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 30, "返回数量")
	cmd.Flags().IntVar(&offset, "offset", 0, "分页偏移")
	cmd.Flags().BoolVar(&asJSON, "json", false, "输出 JSON")
	return cmd
}

func makeClient(opts *rootOptions) (*ncm.Client, error) {
	client, _, _, err := makeClientWithSession(opts)
	return client, err
}

func makeClientWithStatePath(opts *rootOptions) (*ncm.Client, string, error) {
	client, statePath, _, err := makeClientWithSession(opts)
	return client, statePath, err
}

func makeClientWithSession(opts *rootOptions) (*ncm.Client, string, string, error) {
	paths, err := config.Resolve(opts.configDir)
	if err != nil {
		return nil, "", "", err
	}
	statePath, compat, err := config.ExistingStorageState(paths)
	if err != nil {
		return nil, "", "", err
	}
	state, err := config.LoadStorageState(statePath)
	if err != nil {
		return nil, "", "", err
	}
	client, err := ncm.NewClientFromStorageState(state, opts.timeout)
	if err != nil {
		return nil, "", "", err
	}
	profileDir := paths.ProfileDir
	if compat {
		profileDir, err = filepath.Abs(filepath.Join(".ncm", "chrome-profile"))
		if err != nil {
			return nil, "", "", err
		}
	}
	return client, statePath, profileDir, nil
}

func commandContext(cmd *cobra.Command, opts *rootOptions) (context.Context, context.CancelFunc) {
	timeout := opts.timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return context.WithTimeout(cmd.Context(), timeout)
}

func privilegeByID(privileges []ncm.Privilege) map[int64]ncm.Privilege {
	out := make(map[int64]ncm.Privilege, len(privileges))
	for _, privilege := range privileges {
		out[privilege.ID] = privilege
	}
	return out
}

func formatDuration(ms int64) string {
	if ms <= 0 {
		return ""
	}
	seconds := ms / 1000
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func formatPrivilege(priv ncm.Privilege) string {
	if priv.ID == 0 {
		return ""
	}
	if priv.PL > 0 {
		return "yes"
	}
	return "no"
}
