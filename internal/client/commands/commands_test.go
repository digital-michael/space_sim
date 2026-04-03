package commands

import (
	"errors"
	"testing"
)

func TestParse_Empty_ReturnsNil(t *testing.T) {
	cmd, err := Parse("")
	if cmd != nil || err != nil {
		t.Errorf("want nil,nil got %v,%v", cmd, err)
	}
}

func TestParse_Comment_ReturnsNil(t *testing.T) {
	cmd, err := Parse("# this is a comment")
	if cmd != nil || err != nil {
		t.Errorf("want nil,nil got %v,%v", cmd, err)
	}
}

func TestParse_Unknown_ReturnsErrUnknownCommand(t *testing.T) {
	_, err := Parse("frobulate")
	var e ErrUnknownCommand
	if !errors.As(err, &e) {
		t.Errorf("want ErrUnknownCommand, got %T: %v", err, err)
	}
}

func TestParse_SetSpeed_Valid(t *testing.T) {
	cmd, err := Parse("setspeed 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ss, ok := cmd.(SetSpeed)
	if !ok {
		t.Fatalf("want SetSpeed, got %T", cmd)
	}
	if ss.SecondsPerSecond != 10 {
		t.Errorf("want 10, got %v", ss.SecondsPerSecond)
	}
}

func TestParse_SetSpeed_CaseInsensitive(t *testing.T) {
	cmd, err := Parse("SETSPEED 5.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ss, ok := cmd.(SetSpeed)
	if !ok {
		t.Fatalf("want SetSpeed, got %T", cmd)
	}
	if ss.SecondsPerSecond != 5.5 {
		t.Errorf("want 5.5, got %v", ss.SecondsPerSecond)
	}
}

func TestParse_SetSpeed_NonPositive_ReturnsErrUsage(t *testing.T) {
	for _, input := range []string{"setspeed 0", "setspeed -1", "setspeed abc"} {
		_, err := Parse(input)
		var e ErrUsage
		if !errors.As(err, &e) {
			t.Errorf("%q: want ErrUsage, got %T: %v", input, err, err)
		}
	}
}

func TestParse_SetSpeed_MissingArg_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("setspeed")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

func TestParse_GetSpeed(t *testing.T) {
	cmd, err := Parse("getspeed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(GetSpeed); !ok {
		t.Errorf("want GetSpeed, got %T", cmd)
	}
}

func TestParse_SetDataset_AllLevels(t *testing.T) {
	levels := []string{"small", "medium", "large", "huge"}
	for _, level := range levels {
		cmd, err := Parse("setdataset " + level)
		if err != nil {
			t.Fatalf("setdataset %s: unexpected error: %v", level, err)
		}
		sd, ok := cmd.(SetDataset)
		if !ok {
			t.Fatalf("want SetDataset, got %T", cmd)
		}
		if sd.Level != level {
			t.Errorf("level: want %s, got %s", level, sd.Level)
		}
	}
}

func TestParse_SetDataset_InvalidLevel_ReturnsErrUsage(t *testing.T) {
	_, err := Parse("setdataset galactic")
	var e ErrUsage
	if !errors.As(err, &e) {
		t.Errorf("want ErrUsage, got %T: %v", err, err)
	}
}

func TestParse_GetDataset(t *testing.T) {
	cmd, err := Parse("getdataset")
	if err != nil || cmd == nil {
		t.Fatalf("want GetDataset cmd, got err=%v cmd=%v", err, cmd)
	}
	if _, ok := cmd.(GetDataset); !ok {
		t.Errorf("want GetDataset, got %T", cmd)
	}
}

func TestParse_GetTime(t *testing.T) {
	cmd, err := Parse("gettime")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(GetTime); !ok {
		t.Errorf("want GetTime, got %T", cmd)
	}
}

func TestParse_Stream(t *testing.T) {
	cmd, err := Parse("stream")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Stream); !ok {
		t.Errorf("want Stream, got %T", cmd)
	}
}

func TestParse_Help(t *testing.T) {
	cmd, err := Parse("help")
	if err != nil || cmd == nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(Help); !ok {
		t.Errorf("want Help, got %T", cmd)
	}
}

func TestParse_Quit(t *testing.T) {
	for _, input := range []string{"quit", "exit", "QUIT", "EXIT"} {
		cmd, err := Parse(input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", input, err)
		}
		if _, ok := cmd.(Quit); !ok {
			t.Errorf("%q: want Quit, got %T", input, cmd)
		}
	}
}

func TestParse_LeadingTrailingWhitespace(t *testing.T) {
	cmd, err := Parse("  getspeed  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cmd.(GetSpeed); !ok {
		t.Errorf("want GetSpeed, got %T", cmd)
	}
}
