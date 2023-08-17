package measures

type MountPoint struct {
	Name        string  `json:"name"`
	FreeBytes   uint64  `json:"free_b"`
	TotalBytes  uint64  `json:"total_b"`
	FreePercent float64 `json:"free_percent"`
	UsedPercent float64 `json:"used_percent"`
}

func (mp MountPoint) CalcFreePercent() (p float64) {
	if mp.TotalBytes == 0 {
		return 0.0
	}
	return 100.0 - (float64(mp.FreeBytes) / float64(mp.TotalBytes) * 100)
}

func (mp MountPoint) CalcUsedPercent() (p float64) {
	if mp.TotalBytes == 0 {
		return 100.0
	}
	return float64(mp.FreeBytes) / float64(mp.TotalBytes) * 100
}
