package clickhouse

import "testing"

func TestCSVPreview(t *testing.T) {
	result, err := PreviewImportData("id,value\n1,\"comma, quote \"\" and\nnewline\"\n2,plain", "CSVWithNames")
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(result.Rows) != 2 || result.Headers[1] != "value" || result.Rows[0][1] != "comma, quote \" and\nnewline" {
		t.Fatalf("CSV records were split incorrectly: %+v", result)
	}
	if _, err := PreviewImportData("1,\"unterminated", "CSV"); err == nil {
		t.Fatal("malformed CSV accepted")
	}
}
