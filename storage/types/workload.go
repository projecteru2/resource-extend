package types

import (
	"cmp"
	"slices"

	"github.com/cockroachdb/errors"
	resourcetypes "github.com/projecteru2/core/resource/types"
	"github.com/projecteru2/core/utils"
)

type WorkloadResource struct {
	VolumesRequest VolumeBindings `json:"volumes_request"`
	VolumesLimit   VolumeBindings `json:"volumes_limit"`

	VolumePlanRequest VolumePlan `json:"volume_plan_request"`
	VolumePlanLimit   VolumePlan `json:"volume_plan_limit"`

	StorageRequest int64 `json:"storage_request"`
	StorageLimit   int64 `json:"storage_limit"`

	DisksRequest Disks `json:"disks_request"`
	DisksLimit   Disks `json:"disks_limit"`
}

func (w *WorkloadResource) AsRawParams() resourcetypes.RawParams {
	return resourcetypes.RawParams{
		"volumes_request":     w.VolumesRequest,
		"volumes_limit":       w.VolumesLimit,
		"volume_plan_request": w.VolumePlanRequest,
		"volume_plan_limit":   w.VolumePlanLimit,
		"storage_request":     w.StorageRequest,
		"storage_limit":       w.StorageLimit,
		"disks_request":       w.DisksRequest,
		"disks_limit":         w.DisksLimit,
	}
}

func (w *WorkloadResource) Parse(rawParams resourcetypes.RawParams) error {
	return resourcetypes.Decode(rawParams, w)
}

type WorkloadResourceRequest struct {
	VolumesRequest VolumeBindings `json:"volumes_request"`
	VolumesLimit   VolumeBindings `json:"volumes_limit"`
	StorageRequest int64          `json:"storage_request"`
	StorageLimit   int64          `json:"storage_limit"`

	storageFolded bool
}

func (w *WorkloadResourceRequest) Validate() error {
	if len(w.VolumesRequest) == 0 && len(w.VolumesLimit) == 0 && w.StorageLimit == 0 && w.StorageRequest == 0 {
		return nil
	}
	return errors.CombineErrors(
		w.validateVolumes(),
		w.validateStorage(),
	)
}

func (w *WorkloadResourceRequest) Parse(rawParams resourcetypes.RawParams) (err error) {
	if w.VolumesRequest, err = NewVolumeBindings(rawParams.OneOfStringSlice("volumes-request", "volume-request")); err != nil {
		return err
	}
	if w.VolumesLimit, err = NewVolumeBindings(rawParams.OneOfStringSlice("volumes", "volume", "volume-limit", "volumes-limit")); err != nil {
		return err
	}

	if w.StorageRequest, err = utils.ParseRAMInHuman(rawParams.String("storage-request")); err != nil {
		return err
	}
	if w.StorageLimit, err = utils.ParseRAMInHuman(rawParams.String("storage-limit")); err != nil {
		return err
	}
	if rawParams.IsSet("storage") {
		storage, err := utils.ParseRAMInHuman(rawParams.String("storage"))
		if err != nil {
			return err
		}
		w.StorageLimit = storage
		w.StorageRequest = storage
	}

	if len(w.VolumesLimit) > 0 && len(w.VolumesRequest) == 0 {
		w.VolumesRequest = w.VolumesLimit
	}
	return nil
}

// SkipAddStorage will skip adding volume size to storage request / limit (used in realloc)
func (w *WorkloadResourceRequest) SkipAddStorage() {
	w.storageFolded = true
}

func (w *WorkloadResourceRequest) validateVolumes() error {
	if len(w.VolumesRequest) != len(w.VolumesLimit) {
		return errors.Wrap(ErrInvalidVolume, "different length of request and limit")
	}
	if err := w.VolumesRequest.Validate(); err != nil {
		return errors.CombineErrors(ErrInvalidVolume, err)
	}
	if err := w.VolumesLimit.Validate(); err != nil {
		return errors.CombineErrors(ErrInvalidVolume, err)
	}

	slices.SortFunc(w.VolumesRequest, compareBindingString)
	slices.SortFunc(w.VolumesLimit, compareBindingString)

	for i := range w.VolumesRequest {
		request := w.VolumesRequest[i]
		limit := w.VolumesLimit[i]
		if request.Source != limit.Source || request.Destination != limit.Destination || request.Flags != limit.Flags {
			return errors.Wrap(ErrInvalidVolume, "request and limit not match")
		}
		if request.SizeInBytes > 0 && limit.SizeInBytes > 0 && request.SizeInBytes > limit.SizeInBytes {
			limit.SizeInBytes = request.SizeInBytes
		}
		limit.ReadIOPS = max(limit.ReadIOPS, request.ReadIOPS)
		limit.WriteIOPS = max(limit.WriteIOPS, request.WriteIOPS)
		limit.ReadBPS = max(limit.ReadBPS, request.ReadBPS)
		limit.WriteBPS = max(limit.WriteBPS, request.WriteBPS)
	}

	for _, vb := range slices.Concat(w.VolumesRequest, w.VolumesLimit) {
		if err := vb.Validate(); err != nil {
			return errors.CombineErrors(ErrInvalidVolume, err)
		}
	}
	return nil
}

func (w *WorkloadResourceRequest) validateStorage() error {
	if w.StorageLimit < 0 || w.StorageRequest < 0 {
		return errors.Wrap(ErrInvalidStorage, "storage limit or request less than 0")
	}
	w.StorageRequest = cmp.Or(w.StorageRequest, w.StorageLimit)
	if w.StorageLimit > 0 && w.StorageRequest > 0 && w.StorageRequest > w.StorageLimit {
		w.StorageLimit = w.StorageRequest // soft limit storage size
	}

	if !w.storageFolded {
		w.storageFolded = true
		w.StorageRequest += w.VolumesRequest.TotalSize()
		w.StorageLimit += w.VolumesLimit.TotalSize()
	}
	return nil
}

func compareBindingString(b, b1 *VolumeBinding) int {
	return cmp.Compare(b.ToString(false), b1.ToString(false))
}
