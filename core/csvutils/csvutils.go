package csvutils

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
)

func OpenFile(filePath string) (*csv.Reader, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0444)
	if err != nil {
		return nil, err
	}

	csvReader := csv.NewReader(file)
	// identify fields per record and by pass first metadata line.
	record, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	csvReader.FieldsPerRecord = len(record)
	csvReader.ReuseRecord = false
	csvReader.Comma = ','

	return csvReader, nil
}

func ReadAll(reader *csv.Reader) ([][]string, error) {
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

func ReadRows(reader *csv.Reader, n int) ([][]string, []string,error) {
	var records = make([][]string, 0, n)
	for i := 0; i < n; i++ {
		record, err := reader.Read()
		if err != nil && errors.Is(err,io.EOF){
			return records,nil,err
		}else if err != nil && errors.Is(err,csv.ErrFieldCount){
			return records[:len(records)-1],records[len(records)-1],nil
		}
		records = append(records, record)
	}
	return records, nil,nil
}
