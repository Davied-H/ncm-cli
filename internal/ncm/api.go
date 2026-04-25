package ncm

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

type AccountResponse struct {
	Code    int      `json:"code"`
	Profile *Profile `json:"profile"`
}

type Profile struct {
	UserID    int64  `json:"userId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

type PlaylistListResponse struct {
	Code     int        `json:"code"`
	More     bool       `json:"more"`
	Playlist []Playlist `json:"playlist"`
}

type Playlist struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	UserID      int64  `json:"userId"`
	TrackCount  int    `json:"trackCount"`
	Subscribed  bool   `json:"subscribed"`
	SpecialType int    `json:"specialType"`
	Privacy     int    `json:"privacy"`
	CoverImgURL string `json:"coverImgUrl"`
	Creator     *User  `json:"creator,omitempty"`
	Tracks      []Song `json:"tracks,omitempty"`
}

type User struct {
	UserID   int64  `json:"userId"`
	Nickname string `json:"nickname"`
}

type PlaylistDetailResponse struct {
	Code       int         `json:"code"`
	Playlist   Playlist    `json:"playlist"`
	Privileges []Privilege `json:"privileges"`
}

type PlaylistWriteResponse struct {
	Code        int       `json:"code"`
	Message     string    `json:"message,omitempty"`
	Msg         string    `json:"msg,omitempty"`
	Playlist    *Playlist `json:"playlist,omitempty"`
	CoverImgURL string    `json:"coverImgUrl,omitempty"`
}

func (r *PlaylistWriteResponse) MessageText() string {
	if r == nil {
		return ""
	}
	if r.Message != "" {
		return r.Message
	}
	return r.Msg
}

type SongDetailResponse struct {
	Code       int         `json:"code"`
	Songs      []Song      `json:"songs"`
	Privileges []Privilege `json:"privileges"`
}

type Song struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Artists   []Artist   `json:"ar,omitempty"`
	Album     Album      `json:"al,omitempty"`
	Duration  int64      `json:"dt,omitempty"`
	Fee       int        `json:"fee,omitempty"`
	Privilege *Privilege `json:"privilege,omitempty"`
}

func (s *Song) UnmarshalJSON(data []byte) error {
	type rawSong struct {
		ID        int64      `json:"id"`
		Name      string     `json:"name"`
		Ar        []Artist   `json:"ar"`
		Artists   []Artist   `json:"artists"`
		Al        Album      `json:"al"`
		Album     Album      `json:"album"`
		DT        int64      `json:"dt"`
		Duration  int64      `json:"duration"`
		Fee       int        `json:"fee"`
		Privilege *Privilege `json:"privilege"`
	}
	var raw rawSong
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.ID = raw.ID
	s.Name = raw.Name
	s.Artists = raw.Ar
	if len(s.Artists) == 0 {
		s.Artists = raw.Artists
	}
	s.Album = raw.Al
	if s.Album.Name == "" {
		s.Album = raw.Album
	}
	s.Duration = raw.DT
	if s.Duration == 0 {
		s.Duration = raw.Duration
	}
	s.Fee = raw.Fee
	s.Privilege = raw.Privilege
	return nil
}

type Artist struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Album struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Privilege struct {
	ID    int64 `json:"id"`
	Fee   int   `json:"fee"`
	PL    int   `json:"pl"`
	DL    int   `json:"dl"`
	MaxBR int   `json:"maxbr"`
}

type LyricResponse struct {
	Code   int        `json:"code"`
	LRC    LyricBlock `json:"lrc"`
	TLyric LyricBlock `json:"tlyric"`
}

type LyricBlock struct {
	Version int    `json:"version"`
	Lyric   string `json:"lyric"`
}

type SuggestResponse struct {
	Code   int           `json:"code"`
	Result SuggestResult `json:"result"`
}

type SuggestResult struct {
	Songs   []Song   `json:"songs,omitempty"`
	Artists []Artist `json:"artists,omitempty"`
	Albums  []Album  `json:"albums,omitempty"`
	Order   []string `json:"order,omitempty"`
}

type SearchSongResponse struct {
	Code   int              `json:"code"`
	Result SearchSongResult `json:"result"`
}

type SearchSongResult struct {
	SongCount int    `json:"songCount"`
	Songs     []Song `json:"songs"`
}

type SearchPlaylistResponse struct {
	Code   int                  `json:"code"`
	Result SearchPlaylistResult `json:"result"`
}

type SearchPlaylistResult struct {
	PlaylistCount int        `json:"playlistCount"`
	Playlists     []Playlist `json:"playlists"`
}

func (c *Client) Me(ctx context.Context) (*AccountResponse, error) {
	var out AccountResponse
	err := c.WeAPI(ctx, "/api/w/nuser/account/get", nil, &out)
	return &out, err
}

func (c *Client) PlaylistList(ctx context.Context, uid int64, limit, offset int) (*PlaylistListResponse, error) {
	var out PlaylistListResponse
	err := c.WeAPI(ctx, "/api/user/playlist", map[string]any{
		"uid":          uid,
		"limit":        limit,
		"offset":       offset,
		"includeVideo": true,
	}, &out)
	return &out, err
}

func (c *Client) PlaylistDetail(ctx context.Context, id int64, limit int) (*PlaylistDetailResponse, error) {
	var out PlaylistDetailResponse
	err := c.WeAPI(ctx, "/api/v6/playlist/detail", map[string]any{
		"id": id,
		"n":  limit,
		"s":  8,
	}, &out)
	return &out, err
}

func (c *Client) PlaylistCreate(ctx context.Context, name string, private bool, checkToken string) (*PlaylistWriteResponse, error) {
	payload := map[string]any{
		"name":       name,
		"checkToken": checkToken,
	}
	if private {
		payload["privacy"] = 10
	}
	var out PlaylistWriteResponse
	err := c.WeAPI(ctx, "/api/playlist/create", payload, &out)
	return &out, err
}

func (c *Client) PlaylistAddTracks(ctx context.Context, playlistID int64, songIDs []int64) (*PlaylistWriteResponse, error) {
	return c.playlistManipulateTracks(ctx, "add", playlistID, songIDs)
}

func (c *Client) PlaylistRemoveTracks(ctx context.Context, playlistID int64, songIDs []int64) (*PlaylistWriteResponse, error) {
	return c.playlistManipulateTracks(ctx, "del", playlistID, songIDs)
}

func (c *Client) playlistManipulateTracks(ctx context.Context, op string, playlistID int64, songIDs []int64) (*PlaylistWriteResponse, error) {
	trackIDs, err := json.Marshal(songIDs)
	if err != nil {
		return nil, err
	}
	var out PlaylistWriteResponse
	err = c.WeAPI(ctx, "/api/playlist/manipulate/tracks", map[string]any{
		"op":       op,
		"pid":      playlistID,
		"trackIds": string(trackIDs),
		"imme":     true,
	}, &out)
	return &out, err
}

func (c *Client) PlaylistRename(ctx context.Context, id int64, name string) (*PlaylistWriteResponse, error) {
	var out PlaylistWriteResponse
	err := c.WeAPI(ctx, "/api/playlist/update/name", map[string]any{
		"id":   id,
		"name": name,
	}, &out)
	return &out, err
}

func (c *Client) PlaylistUpdateTags(ctx context.Context, id int64, tags []string) (*PlaylistWriteResponse, error) {
	var out PlaylistWriteResponse
	err := c.WeAPI(ctx, "/api/playlist/tags/update", map[string]any{
		"id":   id,
		"tags": strings.Join(tags, ","),
	}, &out)
	return &out, err
}

func (c *Client) PlaylistUpdateDescription(ctx context.Context, id int64, desc string, checkToken string) (*PlaylistWriteResponse, error) {
	var out PlaylistWriteResponse
	err := c.WeAPI(ctx, "/api/playlist/desc/update", map[string]any{
		"id":         id,
		"desc":       desc,
		"checkToken": checkToken,
	}, &out)
	return &out, err
}

func (c *Client) PlaylistDelete(ctx context.Context, id int64) (*PlaylistWriteResponse, error) {
	var out PlaylistWriteResponse
	err := c.WeAPI(ctx, "/api/playlist/delete", map[string]any{
		"pid": id,
	}, &out)
	return &out, err
}

func (c *Client) SongDetail(ctx context.Context, id int64) (*SongDetailResponse, error) {
	cValue, _ := json.Marshal([]map[string]int64{{"id": id}})
	var out SongDetailResponse
	err := c.WeAPI(ctx, "/api/v3/song/detail", map[string]any{
		"c": string(cValue),
	}, &out)
	return &out, err
}

func (c *Client) Lyric(ctx context.Context, id int64) (*LyricResponse, error) {
	var out LyricResponse
	err := c.WeAPI(ctx, "/api/song/lyric", map[string]any{
		"id": id,
		"lv": -1,
		"kv": -1,
		"tv": -1,
	}, &out)
	return &out, err
}

func (c *Client) SearchSuggest(ctx context.Context, keyword string) (*SuggestResponse, error) {
	var out SuggestResponse
	err := c.WeAPI(ctx, "/api/search/suggest/web", map[string]any{
		"s":     keyword,
		"limit": 8,
	}, &out)
	return &out, err
}

func (c *Client) SearchSongs(ctx context.Context, keyword string, limit, offset int) (*SearchSongResponse, error) {
	var out SearchSongResponse
	err := c.WeAPI(ctx, "/api/cloudsearch/get/web", cloudSearchPayload(keyword, 1, limit, offset), &out)
	return &out, err
}

func (c *Client) SearchPlaylists(ctx context.Context, keyword string, limit, offset int) (*SearchPlaylistResponse, error) {
	var out SearchPlaylistResponse
	err := c.WeAPI(ctx, "/api/cloudsearch/get/web", cloudSearchPayload(keyword, 1000, limit, offset), &out)
	return &out, err
}

func cloudSearchPayload(keyword string, typ, limit, offset int) map[string]any {
	return map[string]any{
		"s":         keyword,
		"type":      typ,
		"limit":     limit,
		"offset":    offset,
		"total":     true,
		"hlpretag":  `<span class="s-fc7">`,
		"hlposttag": "</span>",
	}
}

func ArtistsText(artists []Artist) string {
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		if artist.Name != "" {
			names = append(names, artist.Name)
		}
	}
	return strings.Join(names, "/")
}

func ParseID(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}
