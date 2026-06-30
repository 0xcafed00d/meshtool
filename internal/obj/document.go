package obj

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Document struct {
	Records []Record
}

type Record struct {
	Kind          string
	Raw           string
	InlineComment string
	Vertex        *VertexRecord
	TexCoord      *TexCoordRecord
	Normal        *NormalRecord
	Face          *FaceRecord
}

type VertexRecord struct {
	Position Vec3
	Extra    []string
}

type TexCoordRecord struct {
	Values []float64
}

type NormalRecord struct {
	Direction Vec3
	Extra     []string
}

type FaceRecord struct {
	Refs []string
}

func Parse(reader io.Reader) (*Document, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	doc := &Document{}
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSuffix(scanner.Text(), "\r")
		record, err := parseRecord(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		doc.Records = append(doc.Records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return doc, nil
}

func parseRecord(line string) (Record, error) {
	record := Record{Raw: line}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return record, nil
	}

	head, comment := splitInlineComment(line)
	fields := strings.Fields(head)
	if len(fields) == 0 {
		return record, nil
	}

	record.Kind = fields[0]
	record.InlineComment = comment

	switch record.Kind {
	case "v":
		if len(fields) < 4 {
			return record, fmt.Errorf("vertex requires x y z")
		}
		position, err := parseVec3(fields[1], fields[2], fields[3])
		if err != nil {
			return record, fmt.Errorf("vertex: %w", err)
		}
		record.Vertex = &VertexRecord{
			Position: position,
			Extra:    cloneStrings(fields[4:]),
		}
	case "vt":
		if len(fields) < 2 {
			return record, fmt.Errorf("texture coordinate requires at least u")
		}
		values, err := parseFloatFields(fields[1:])
		if err != nil {
			return record, fmt.Errorf("texture coordinate: %w", err)
		}
		record.TexCoord = &TexCoordRecord{Values: values}
	case "vn":
		if len(fields) < 4 {
			return record, fmt.Errorf("normal requires x y z")
		}
		direction, err := parseVec3(fields[1], fields[2], fields[3])
		if err != nil {
			return record, fmt.Errorf("normal: %w", err)
		}
		record.Normal = &NormalRecord{
			Direction: direction,
			Extra:     cloneStrings(fields[4:]),
		}
	case "f":
		if len(fields) < 4 {
			return record, fmt.Errorf("face requires at least 3 vertex references")
		}
		record.Face = &FaceRecord{Refs: cloneStrings(fields[1:])}
	}

	return record, nil
}

func splitInlineComment(line string) (string, string) {
	index := strings.Index(line, "#")
	if index < 0 {
		return line, ""
	}
	return line[:index], strings.TrimSpace(line[index:])
}

func parseVec3(xValue string, yValue string, zValue string) (Vec3, error) {
	x, err := strconv.ParseFloat(xValue, 64)
	if err != nil {
		return Vec3{}, err
	}
	y, err := strconv.ParseFloat(yValue, 64)
	if err != nil {
		return Vec3{}, err
	}
	z, err := strconv.ParseFloat(zValue, 64)
	if err != nil {
		return Vec3{}, err
	}
	return Vec3{X: x, Y: y, Z: z}, nil
}

func (doc *Document) Write(writer io.Writer) error {
	buffered := bufio.NewWriter(writer)
	for _, record := range doc.Records {
		if _, err := buffered.WriteString(record.Format()); err != nil {
			return err
		}
		if _, err := buffered.WriteString("\n"); err != nil {
			return err
		}
	}
	return buffered.Flush()
}

func (record Record) Format() string {
	switch {
	case record.Vertex != nil:
		return formatVertex("v", record.Vertex.Position, record.Vertex.Extra, record.InlineComment)
	case record.TexCoord != nil:
		return formatTexCoord(record.TexCoord.Values, record.InlineComment)
	case record.Normal != nil:
		return formatVertex("vn", record.Normal.Direction, record.Normal.Extra, record.InlineComment)
	case record.Face != nil:
		line := "f " + strings.Join(record.Face.Refs, " ")
		if record.InlineComment != "" {
			line += " " + record.InlineComment
		}
		return line
	default:
		return record.Raw
	}
}

func formatVertex(kind string, vector Vec3, extra []string, comment string) string {
	parts := []string{
		kind,
		formatFloat(vector.X),
		formatFloat(vector.Y),
		formatFloat(vector.Z),
	}
	parts = append(parts, extra...)
	line := strings.Join(parts, " ")
	if comment != "" {
		line += " " + comment
	}
	return line
}

func formatTexCoord(values []float64, comment string) string {
	parts := []string{"vt"}
	for _, value := range values {
		parts = append(parts, formatFloat(value))
	}
	line := strings.Join(parts, " ")
	if comment != "" {
		line += " " + comment
	}
	return line
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(cleanFloat(value), 'g', -1, 64)
}

func parseFloatFields(fields []string) ([]float64, error) {
	values := make([]float64, len(fields))
	for index, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return nil, err
		}
		values[index] = value
	}
	return values, nil
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	clone := make([]string, len(values))
	copy(clone, values)
	return clone
}
