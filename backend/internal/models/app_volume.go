package models

type AppVolumeModel struct {
	VolumeID     string   `json:"volumeID" gorm:"column:id"`
	AppID        string   `json:"appID"`
	Slug         string   `json:"slug"`
	MountPath    string   `json:"mountPath"`
	SubPath      string   `json:"subPath,omitempty"` // Optional sub-path within the mount
	VolumeType   string   `json:"volumeType"`
	Capacity     int      `json:"capacity"`
	AccessModes  []string `json:"accessModes"`
	StorageClass string   `json:"storageClass"`
	VolumeMode   string   `json:"volumeMode"`
}

type ListAppVolumesRequest struct {
	AppID string `uri:"appID"`
}

type ListAppVolumesResponse struct {
	Total   int64             `json:"total"`
	Records []*AppVolumeModel `json:"records"`
}

type CreateAppVolumeRequest struct {
	AppID        string   `json:"-" uri:"appID"`
	Slug         string   `json:"slug" binding:"required"`
	MountPath    string   `json:"mountPath" binding:"required"`
	SubPath      string   `json:"subPath"`
	Capacity     int      `json:"capacity" binding:"required"`
	VolumeType   string   `json:"volumeType" binding:"required"`
	AccessModes  []string `json:"accessModes" binding:"required"`
	StorageClass string   `json:"storageClass"`
	VolumeMode   string   `json:"volumeMode" binding:"required,oneof=Filesystem Block"`
}

type UpdateAppVolumeRequest struct {
	AppID     string `json:"-" uri:"appID"`
	VolumeID  string `json:"-" uri:"volumeID"`
	MountPath string `json:"mountPath" binding:"required"`
	SubPath   string `json:"subPath"`
}

type DeleteAppVolumesRequest struct {
	AppID     string   `json:"-" uri:"appID"`
	VolumeIDs []string `json:"volumeIDs" binding:"required,dive,required"` // List of volume IDs to delete
}
