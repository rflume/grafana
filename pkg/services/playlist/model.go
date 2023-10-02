package playlist

import (
	"errors"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/grafana/grafana/pkg/kinds/playlist"
)

// Typed errors
var (
	ErrPlaylistNotFound                = errors.New("Playlist not found")
	ErrPlaylistFailedGenerateUniqueUid = errors.New("failed to generate unique playlist UID")
	ErrCommandValidationFailed         = errors.New("command missing required fields")
)

// Playlist model
type Playlist struct {
	Id       int64  `json:"id,omitempty" db:"id"`
	UID      string `json:"uid" xorm:"uid" db:"uid"`
	Name     string `json:"name" db:"name"`
	Interval string `json:"interval" db:"interval"`
	OrgId    int64  `json:"-" db:"org_id"`

	// Added for kubernetes migration + synchronization
	ItemJSON  []byte    `json:"-" db:"item_json"`
	CreatedAt time.Time `json:"createdAt,omitempty" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" db:"updated_at"`
}

type PlaylistDTO = playlist.Spec
type PlaylistItemDTO = playlist.Item
type PlaylistItemType = playlist.ItemType

type PlaylistItem struct {
	Id         int64  `json:"-" db:"id"`
	PlaylistId int64  `json:"-" db:"playlist_id"`
	Type       string `json:"type" db:"type"`
	Value      string `json:"value" db:"value"`
	Order      int    `json:"-" db:"order"`
	Title      string `json:"title,omitempty" db:"title"`
}

type Playlists []*Playlist

//
// COMMANDS
//

type UpdatePlaylistCommand struct {
	OrgId    int64          `json:"-"`
	UID      string         `json:"uid"`
	Name     string         `json:"name" binding:"Required"`
	Interval string         `json:"interval"`
	Items    []PlaylistItem `json:"items"`
}

type CreatePlaylistCommand struct {
	Name     string         `json:"name" binding:"Required"`
	Interval string         `json:"interval"`
	Items    []PlaylistItem `json:"items"`
	OrgId    int64          `json:"-"`
}

type DeletePlaylistCommand struct {
	UID   string
	OrgId int64
}

//
// QUERIES
//

type GetPlaylistsQuery struct {
	// NOTE: the frontend never sends this query
	Name  string
	Limit int
	OrgId int64
}

type GetPlaylistByUidQuery struct {
	UID   string
	OrgId int64
}

type GetPlaylistItemsByUidQuery struct {
	PlaylistUID string
	OrgId       int64
}

func ConvertToK8sResource(v *Playlist, items []PlaylistItemDTO) *playlist.Playlist {
	return &playlist.Playlist{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Playlist",
			APIVersion: playlist.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              v.UID,
			UID:               types.UID(v.UID), // required for k8s update function
			ResourceVersion:   fmt.Sprintf("%d", v.UpdatedAt.UnixMilli()),
			CreationTimestamp: metav1.NewTime(v.CreatedAt),
			Namespace:         fmt.Sprintf("org-%d", v.OrgId),
			Annotations: map[string]string{
				"grafana.com/updatedTime": v.UpdatedAt.Format(time.RFC3339),
			},
		},
		Spec: &playlist.Spec{
			Uid:      v.UID,
			Name:     v.Name,
			Interval: v.Interval,
			Items:    items,
		},
	}
}
