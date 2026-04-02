package huawei

import (
	"testing"
	"time"
)

func TestParsePrevueListPrograms(t *testing.T) {
	t.Run("normal response with multiple programs", func(t *testing.T) {
		data := []byte(`{
			"totalSize": 3,
			"curPage": 1,
			"totalPage": 1,
			"channelName": "榆林一套高清",
			"channelPrevueList": [
				{
					"prevueName": "测试卡信号",
					"contentName": "测试卡信号",
					"startTime": "01:15:00",
					"endTime": "07:00:00",
					"currDate": "2025-02-15",
					"channelID": "20000246",
					"isLive": "0",
					"isBack": "1",
					"isFuture": "0"
				},
				{
					"prevueName": "中华人民共和国国歌",
					"contentName": "中华人民共和国国歌",
					"startTime": "07:00:00",
					"endTime": "07:01:00",
					"currDate": "2025-02-15",
					"channelID": "20000246",
					"isLive": "0",
					"isBack": "1",
					"isFuture": "0"
				},
				{
					"prevueName": "榆林新闻联播",
					"contentName": "榆林新闻联播",
					"startTime": "07:01:00",
					"endTime": "07:15:00",
					"currDate": "2025-02-15",
					"channelID": "20000246",
					"isLive": "0",
					"isBack": "1",
					"isFuture": "0"
				}
			]
		}`)

		programs, err := parsePrevueListPrograms(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(programs) != 3 {
			t.Fatalf("expected 3 programs, got %d", len(programs))
		}

		// Verify first program
		if programs[0].Title != "测试卡信号" {
			t.Errorf("expected title '测试卡信号', got '%s'", programs[0].Title)
		}

		loc := time.Local
		expectedStart := time.Date(2025, 2, 15, 1, 15, 0, 0, loc)
		if !programs[0].StartTime.Equal(expectedStart) {
			t.Errorf("expected start time %v, got %v", expectedStart, programs[0].StartTime)
		}

		expectedEnd := time.Date(2025, 2, 15, 7, 0, 0, 0, loc)
		if !programs[0].EndTime.Equal(expectedEnd) {
			t.Errorf("expected end time %v, got %v", expectedEnd, programs[0].EndTime)
		}
	})

	t.Run("cross-midnight program", func(t *testing.T) {
		data := []byte(`{
			"totalSize": 1,
			"curPage": 1,
			"totalPage": 1,
			"channelName": "Test Channel",
			"channelPrevueList": [
				{
					"prevueName": "健康一家人",
					"contentName": "健康一家人",
					"startTime": "23:20:00",
					"endTime": "01:05:00",
					"currDate": "2025-02-15",
					"channelID": "20000246",
					"isLive": "1",
					"isBack": "0",
					"isFuture": "0"
				}
			]
		}`)

		programs, err := parsePrevueListPrograms(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(programs) != 1 {
			t.Fatalf("expected 1 program, got %d", len(programs))
		}

		loc := time.Local
		expectedStart := time.Date(2025, 2, 15, 23, 20, 0, 0, loc)
		expectedEnd := time.Date(2025, 2, 16, 1, 5, 0, 0, loc)

		if !programs[0].StartTime.Equal(expectedStart) {
			t.Errorf("expected start time %v, got %v", expectedStart, programs[0].StartTime)
		}
		if !programs[0].EndTime.Equal(expectedEnd) {
			t.Errorf("expected end time %v, got %v", expectedEnd, programs[0].EndTime)
		}
	})

	t.Run("empty channel prevue list", func(t *testing.T) {
		data := []byte(`{
			"totalSize": 0,
			"curPage": 1,
			"totalPage": 1,
			"channelName": "Empty Channel",
			"channelPrevueList": []
		}`)

		_, err := parsePrevueListPrograms(data)
		if err == nil {
			t.Fatal("expected error for empty list, got nil")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		data := []byte(`not valid json`)

		_, err := parsePrevueListPrograms(data)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})

	t.Run("prevueName fallback to contentName", func(t *testing.T) {
		data := []byte(`{
			"totalSize": 1,
			"curPage": 1,
			"totalPage": 1,
			"channelName": "Test",
			"channelPrevueList": [
				{
					"prevueName": "",
					"contentName": "Fallback Name",
					"startTime": "10:00:00",
					"endTime": "11:00:00",
					"currDate": "2025-02-15",
					"channelID": "20000246",
					"isLive": "0",
					"isBack": "1",
					"isFuture": "0"
				}
			]
		}`)

		programs, err := parsePrevueListPrograms(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(programs) != 1 {
			t.Fatalf("expected 1 program, got %d", len(programs))
		}
		if programs[0].Title != "Fallback Name" {
			t.Errorf("expected title 'Fallback Name', got '%s'", programs[0].Title)
		}
	})

	t.Run("skip programs with missing date or time", func(t *testing.T) {
		data := []byte(`{
			"totalSize": 3,
			"curPage": 1,
			"totalPage": 1,
			"channelName": "Test",
			"channelPrevueList": [
				{
					"prevueName": "Valid",
					"startTime": "10:00:00",
					"endTime": "11:00:00",
					"currDate": "2025-02-15",
					"channelID": "20000246"
				},
				{
					"prevueName": "Missing Date",
					"startTime": "10:00:00",
					"endTime": "11:00:00",
					"currDate": "",
					"channelID": "20000246"
				},
				{
					"prevueName": "Missing Time",
					"startTime": "",
					"endTime": "11:00:00",
					"currDate": "2025-02-15",
					"channelID": "20000246"
				}
			]
		}`)

		programs, err := parsePrevueListPrograms(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(programs) != 1 {
			t.Fatalf("expected 1 valid program, got %d", len(programs))
		}
		if programs[0].Title != "Valid" {
			t.Errorf("expected title 'Valid', got '%s'", programs[0].Title)
		}
	})
}
