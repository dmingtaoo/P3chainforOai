package viewChange

import "testing"

func TestUpdateConfigFile(t *testing.T) {
	ReloadConfigPath()
	dc := ReloadConfigFile()
	UpdateConfigFile(dc, "Follower")
}
