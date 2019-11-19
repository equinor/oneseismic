package config

var (
	majVer   string
	minVer   string
	patchVer string
)

func SetDevVersion(maj, min, patch string) {
	majVer = maj
	minVer = min
	patchVer = patch
}
