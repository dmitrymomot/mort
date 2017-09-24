package object

import (
	"errors"
	"regexp"
	"strings"

	Logger "github.com/labstack/gommon/log"
	"mort/config"
	"mort/transforms"
)

const (
	URI_TYPE_S3    = 0
	URI_TYPE_LOCAL = 1
)

var URI_LIIP_RE = regexp.MustCompile(`\/media\/cache\/.*`)
var URI_LOCAL_RE = regexp.MustCompile(`\/media\/.*`)

func presetToTransform(preset config.PresetsYaml) transforms.Transforms {
	var trans transforms.Transforms
	filters := preset.Filters

	if len(filters.Thumbnail.Size) > 0 {
		trans.ResizeT(filters.Thumbnail.Size, filters.Thumbnail.Mode == "outbound")
	}

	if len(filters.SmartCrop.Size) > 0 {
		trans.CropT(filters.SmartCrop.Size, filters.SmartCrop.Mode == "outbound")
	}

	if len(filters.Crop.Size) > 0 {
		trans.CropT(filters.Crop.Size, filters.Crop.Mode == "outbound")
	}

	trans.Quality = preset.Quality

	return trans
}

type FileObject struct {
	Uri        string                `json:"uri"`
	Bucket     string                `json:"bucket"`
	Key        string                `json:"key"`
	UriType    int                   `json:"uriType"`
	Parent     string                `json:"parent"`
	Transforms transforms.Transforms `json:"transforms"`
	Storage    config.Storage        `json:"storage"`
}

func NewFileObject(path string) (*FileObject, error) {
	obj := FileObject{}
	obj.Uri = path
	if URI_LOCAL_RE.MatchString(path) {
		obj.UriType = URI_TYPE_LOCAL
	} else {
		obj.UriType = URI_TYPE_S3
	}

	err := obj.decode()
	Logger.Infof("UriType = %d key = %s bucket = %s parent = %s err = %s\n", obj.UriType, obj.Key, obj.Bucket, obj.Parent, err)
	return &obj, err
}

func (self *FileObject) decode() error {
	//if self.UriType == URI_TYPE_LOCAL {
	//	return self.decodeLiipPath()
	//}

	elements := strings.Split(self.Uri, "/")
	if len(elements) < 3 {
		return errors.New("Invalid path")
	}

	self.Bucket = elements[1]
	self.Key = "/" + strings.Join(elements[2:], "/")
	if bucket, ok := config.GetInstance().Buckets[self.Bucket]; ok {
		self.decodeKey(bucket)
		if self.HasTransform() {
			self.Storage = bucket.Storages.Transform
		} else {
			self.Storage = bucket.Storages.Basic
		}

	} else {
		return errors.New("Unknown bucket")
	}

	return nil
}

func (self *FileObject) decodeKey(bucket config.Bucket) error {
	pathRe := regexp.MustCompile(bucket.Transform.Path)
	matches := pathRe.FindStringSubmatch(self.Key)
	if len(matches) == 0 {
		return nil
	}

	Logger.Infof(" m = %s",  matches)
	presetName := string(matches[1])
	parent := "/" + string(matches[2])

	self.Transforms = presetToTransform(bucket.Transform.Presets[presetName])
	self.Parent = parent
	Logger.Infof("uri: %s parent: %s key: %s len: %d \n", self.Uri, self.Parent, self.Key)
	return nil
}

//func (self *FileObject) decodeLiipPath() error {
//	self.Uri = strings.Replace(self.Uri, "//", "/", 3)
//	key := strings.Replace(self.Uri, "/media/cache", "", 1)
//	key = strings.Replace(key, "/resolve", "", 1)
//	elements := strings.Split(key, "/")
//	if URI_LIIP_RE.MatchString(self.Uri) {
//		presetName := elements[1]
//		//self.Key = strings.Replace(self.Uri, "//", "/", 3)
//		self.Key = strings.Replace(self.Uri, "//", "/", 3)
//		self.Parent =  "/" + strings.Join(elements[4:], "/")
//		liipConfig := config.GetInstance().LiipConfig
//		self.Transforms = liipToTransform(liipConfig[presetName])
//	} else {
//		self.Key = self.Uri
//	}
//
//	Logger.Debugf("uri: %s parent: %s key: %s len: %d \n", self.Uri, self.Parent, self.Key, len(elements))
//	return nil
//}

func (self *FileObject) GetParent() *FileObject {
	parent, _ := NewFileObject(self.Parent)
	return parent
}

func (self *FileObject) HasParent() bool {
	return self.Parent != ""
}

func (self *FileObject) HasTransform() bool {
	return self.Transforms.NotEmpty == true
}