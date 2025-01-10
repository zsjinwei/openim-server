// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apistruct

type PictureBaseInfo struct {
	UUID   string `mapstructure:"uuid"   json:"uuid"`
	Type   string `mapstructure:"type"   json:"type"   validate:"required"`
	Size   int64  `mapstructure:"size"   json:"size"`
	Width  int32  `mapstructure:"width"  json:"width"  validate:"required"`
	Height int32  `mapstructure:"height" json:"height" validate:"required"`
	Url    string `mapstructure:"url"    json:"url"    validate:"required"`
}

type PictureElem struct {
	SourcePath      string          `mapstructure:"sourcePath"      json:"sourcePath"`
	SourcePicture   PictureBaseInfo `mapstructure:"sourcePicture"   json:"sourcePicture"   validate:"required"`
	BigPicture      PictureBaseInfo `mapstructure:"bigPicture"      json:"bigPicture"      validate:"required"`
	SnapshotPicture PictureBaseInfo `mapstructure:"snapshotPicture" json:"snapshotPicture" validate:"required"`
}

type SoundElem struct {
	UUID      string `mapstructure:"uuid"      json:"uuid"`
	SoundPath string `mapstructure:"soundPath" json:"soundPath"`
	SourceURL string `mapstructure:"sourceUrl" json:"sourceUrl" validate:"required"`
	DataSize  int64  `mapstructure:"dataSize"  json:"dataSize"`
	Duration  int64  `mapstructure:"duration"  json:"duration"  validate:"required,min=1"`
}

type VideoElem struct {
	VideoPath      string `mapstructure:"videoPath"      json:"videoPath"`
	VideoUUID      string `mapstructure:"videoUUID"      json:"videoUUID"`
	VideoURL       string `mapstructure:"videoUrl"       json:"videoUrl"       validate:"required"`
	VideoType      string `mapstructure:"videoType"      json:"videoType"      validate:"required"`
	VideoSize      int64  `mapstructure:"videoSize"      json:"videoSize"      validate:"required"`
	Duration       int64  `mapstructure:"duration"       json:"duration"       validate:"required"`
	SnapshotPath   string `mapstructure:"snapshotPath"   json:"snapshotPath"`
	SnapshotUUID   string `mapstructure:"snapshotUUID"   json:"snapshotUUID"`
	SnapshotSize   int64  `mapstructure:"snapshotSize"   json:"snapshotSize"`
	SnapshotURL    string `mapstructure:"snapshotUrl"    json:"snapshotUrl"    validate:"required"`
	SnapshotWidth  int32  `mapstructure:"snapshotWidth"  json:"snapshotWidth"  validate:"required"`
	SnapshotHeight int32  `mapstructure:"snapshotHeight" json:"snapshotHeight" validate:"required"`
}

type FileElem struct {
	FilePath  string `mapstructure:"filePath"  json:"filePath"`
	UUID      string `mapstructure:"uuid"      json:"uuid"`
	SourceURL string `mapstructure:"sourceUrl" json:"sourceUrl" validate:"required"`
	FileName  string `mapstructure:"fileName"  json:"fileName"  validate:"required"`
	FileSize  int64  `mapstructure:"fileSize"  json:"fileSize"  validate:"required"`
}
type AtElem struct {
	Text       string   `mapstructure:"text"       json:"text"`
	AtUserList []string `mapstructure:"atUserList" json:"atUserList" validate:"required,max=1000"`
	IsAtSelf   bool     `mapstructure:"isAtSelf"   json:"isAtSelf"`
}
type LocationElem struct {
	Description string  `mapstructure:"description" json:"description"`
	Longitude   float64 `mapstructure:"longitude"   json:"longitude" validate:"required"`
	Latitude    float64 `mapstructure:"latitude"    json:"latitude"  validate:"required"`
}

type CustomElem struct {
	Data        string `mapstructure:"data"        json:"data" validate:"required"`
	Description string `mapstructure:"description" json:"description"`
	Extension   string `mapstructure:"extension"   json:"extension"`
}

type TextElem struct {
	Content string `json:"content" validate:"required"`
}

type RevokeElem struct {
	RevokeMsgClientID string `mapstructure:"revokeMsgClientID" json:"revokeMsgClientID" validate:"required"`
}

type OANotificationElem struct {
	NotificationName    string       `mapstructure:"notificationName"    json:"notificationName"    validate:"required"`
	NotificationFaceURL string       `mapstructure:"notificationFaceURL" json:"notificationFaceURL"`
	NotificationType    int32        `mapstructure:"notificationType"    json:"notificationType"    validate:"required"`
	Text                string       `mapstructure:"text"                json:"text"                validate:"required"`
	Url                 string       `mapstructure:"url"                 json:"url"`
	MixType             int32        `mapstructure:"mixType"             json:"mixType"             validate:"gte=0,lte=5"`
	PictureElem         *PictureElem `mapstructure:"pictureElem"         json:"pictureElem"`
	SoundElem           *SoundElem   `mapstructure:"soundElem"           json:"soundElem"`
	VideoElem           *VideoElem   `mapstructure:"videoElem"           json:"videoElem"`
	FileElem            *FileElem    `mapstructure:"fileElem"            json:"fileElem"`
	Ex                  string       `mapstructure:"ex"                  json:"ex"`
}
type MessageRevoked struct {
	RevokerID       string `mapstructure:"revokerID"       json:"revokerID"       validate:"required"`
	RevokerRole     int32  `mapstructure:"revokerRole"     json:"revokerRole"     validate:"required"`
	ClientMsgID     string `mapstructure:"clientMsgID"     json:"clientMsgID"     validate:"required"`
	RevokerNickname string `mapstructure:"revokerNickname" json:"revokerNickname"`
	SessionType     int32  `mapstructure:"sessionType"     json:"sessionType"     validate:"required"`
	Seq             uint32 `mapstructure:"seq"             json:"seq"             validate:"required"`
}
