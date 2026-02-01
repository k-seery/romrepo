package rom

type Location int

const (
	ServerOnly Location = iota
	OnBoth
)

type ROMStatus struct {
	Name       string
	Location   Location
	ServerSize int64
	ServerPath string
}

func Diff(serverROMs []ROMFile, clientFileNames map[string]bool) []ROMStatus {
	var result []ROMStatus
	for _, sr := range serverROMs {
		loc := ServerOnly
		if clientFileNames[sr.Name] {
			loc = OnBoth
		}
		result = append(result, ROMStatus{
			Name:       sr.Name,
			Location:   loc,
			ServerSize: sr.Size,
			ServerPath: sr.Path,
		})
	}
	return result
}
