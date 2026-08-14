package main

import "testing"

func TestParseCommandJSONPlain(t *testing.T) {
	raw := `[{"device_id":"gym-climat","command":"set_temp","value":18},{"device_id":"pool-light","command":"set_color","value":"0000FF"}]`
	cmds, err := parseCommandJSON(raw)
	if err != nil {
		t.Fatalf("не распарсилось: %v", err)
	}
	if len(cmds) != 2 {
		t.Fatalf("ждём 2 команды, пришло %d", len(cmds))
	}
	if cmds[0].DeviceID != "gym-climat" || cmds[0].Command != "set_temp" || cmds[0].Value != float64(18) {
		t.Fatalf("первая команда не та: %+v", cmds[0])
	}
	if cmds[1].Command != "set_color" || cmds[1].Value != "0000FF" {
		t.Fatalf("вторая команда не та: %+v", cmds[1])
	}
}

func TestParseCommandJSONFenced(t *testing.T) {
	raw := "Конечно, вот команды:\n```json\n[{\"device_id\":\"light-living\",\"command\":\"off\",\"value\":false}]\n```\n"
	cmds, err := parseCommandJSON(raw)
	if err != nil {
		t.Fatalf("не распарсилось: %v", err)
	}
	if len(cmds) != 1 || cmds[0].Command != "off" {
		t.Fatalf("команда не та: %+v", cmds)
	}
}

func TestParseCommandJSONGarbage(t *testing.T) {
	if _, err := parseCommandJSON("просто текст без json"); err == nil {
		t.Fatal("мусор должен давать ошибку")
	}
}
