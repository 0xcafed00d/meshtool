package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"meshtool/internal/obj"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "meshtool:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("missing command")
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	case "info":
		return runInfo(args[1:], stdin, stdout)
	case "transform":
		return runTransform(args[1:], stdin, stdout, stderr)
	case "triangulate":
		return runTriangulate(args[1:], stdin, stdout, stderr)
	case "slice":
		return runSlice(args[1:], stdin, stdout, stderr)
	case "chain":
		return runChain(args[1:], stdin, stdout, stderr)
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runInfo(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet("info", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: meshtool info <input.obj>")
	}

	doc, err := readOBJ(flags.Arg(0), stdin)
	if err != nil {
		return err
	}
	printInfo(stdout, flags.Arg(0), doc.Stats())
	return nil
}

func runTransform(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	flags, values := newTransformFlagSet("transform")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 {
		return errors.New("usage: meshtool transform [options] <input.obj> <output.obj>")
	}

	doc, err := readOBJ(flags.Arg(0), stdin)
	if err != nil {
		return err
	}
	options, err := transformOptionsFromFlags(values)
	if err != nil {
		return err
	}
	if err := doc.Transform(options); err != nil {
		return err
	}
	if err := writeOBJ(flags.Arg(1), stdout, doc); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "wrote %s\n", flags.Arg(1))
	return nil
}

func runTriangulate(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("triangulate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 {
		return errors.New("usage: meshtool triangulate <input.obj> <output.obj>")
	}

	doc, err := readOBJ(flags.Arg(0), stdin)
	if err != nil {
		return err
	}
	result := doc.Triangulate()
	if err := writeOBJ(flags.Arg(1), stdout, doc); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "triangulated %d polygon face(s) into %d triangle face(s); wrote %s\n", result.Polygons, result.NewTriangles, flags.Arg(1))
	return nil
}

func runSlice(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	flags, values := newSliceFlagSet("slice")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 3 {
		return errors.New("usage: meshtool slice [options] <x-|x+|y-|y+|z-|z+> <input.obj> <output.obj>")
	}

	options, err := sliceOptionsFromFlags(values, flags.Arg(0))
	if err != nil {
		return err
	}
	doc, err := readOBJ(flags.Arg(1), stdin)
	if err != nil {
		return err
	}
	result, err := doc.Slice(options)
	if err != nil {
		return err
	}
	if err := writeOBJ(flags.Arg(2), stdout, doc); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "sliced %d face(s) into %d face(s), split %d, discarded %d; wrote %s\n", result.InputFaces, result.OutputFaces, result.SplitFaces, result.DiscardedFaces, flags.Arg(2))
	return nil
}

func runChain(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) < 3 {
		return errors.New("usage: meshtool chain <input.obj> <output.obj> <operation> [operation ...]")
	}

	doc, err := readOBJ(args[0], stdin)
	if err != nil {
		return err
	}

	operations := args[2:]
	applied := 0
	for len(operations) > 0 {
		operation := operations[0]
		operations = operations[1:]

		switch operation {
		case "transform":
			var operationArgs []string
			operationArgs, operations = splitOperationArgs(operations)
			flags, values := newTransformFlagSet("chain transform")
			if err := flags.Parse(operationArgs); err != nil {
				return fmt.Errorf("transform: %w", err)
			}
			if flags.NArg() != 0 {
				return fmt.Errorf("transform: unexpected argument %q", flags.Arg(0))
			}
			options, err := transformOptionsFromFlags(values)
			if err != nil {
				return fmt.Errorf("transform: %w", err)
			}
			if err := doc.Transform(options); err != nil {
				return fmt.Errorf("transform: %w", err)
			}
			applied++
		case "triangulate":
			doc.Triangulate()
			applied++
		case "slice":
			var operationArgs []string
			operationArgs, operations = splitOperationArgs(operations)
			flags, values := newSliceFlagSet("chain slice")
			if err := flags.Parse(operationArgs); err != nil {
				return fmt.Errorf("slice: %w", err)
			}
			if flags.NArg() != 1 {
				return fmt.Errorf("slice: expected one side argument")
			}
			options, err := sliceOptionsFromFlags(values, flags.Arg(0))
			if err != nil {
				return fmt.Errorf("slice: %w", err)
			}
			if _, err := doc.Slice(options); err != nil {
				return fmt.Errorf("slice: %w", err)
			}
			applied++
		default:
			return fmt.Errorf("unknown chain operation %q", operation)
		}
	}

	if err := writeOBJ(args[1], stdout, doc); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "applied %d operation(s); wrote %s\n", applied, args[1])
	return nil
}

func splitOperationArgs(args []string) ([]string, []string) {
	for index, arg := range args {
		if isChainOperation(arg) {
			return args[:index], args[index:]
		}
	}
	return args, nil
}

func isChainOperation(arg string) bool {
	return arg == "transform" || arg == "triangulate" || arg == "slice"
}

func readOBJ(path string, stdin io.Reader) (*obj.Document, error) {
	if path == "-" {
		return obj.Parse(stdin)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return obj.Parse(file)
}

func writeOBJ(path string, stdout io.Writer, doc *obj.Document) error {
	var writer io.Writer
	var file *os.File

	if path == "-" {
		writer = stdout
	} else {
		var err error
		file, err = os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}

	return doc.Write(writer)
}

type transformFlagValues struct {
	uniformScale   *float64
	scaleX         *float64
	scaleY         *float64
	scaleZ         *float64
	translate      *string
	translateX     *float64
	translateY     *float64
	translateZ     *float64
	rotateX        *float64
	rotateY        *float64
	rotateZ        *float64
	matrixValue    *string
	center         *bool
	normalize      *float64
	flipX          *bool
	flipY          *bool
	flipZ          *bool
	reverseWinding *bool
}

type sliceFlagValues struct {
	boundary *float64
	epsilon  *float64
}

func newSliceFlagSet(name string) (*flag.FlagSet, sliceFlagValues) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	values := sliceFlagValues{
		boundary: flags.Float64("at", 0, "slice plane coordinate"),
		epsilon:  flags.Float64("eps", 1e-9, "classification epsilon"),
	}
	return flags, values
}

func sliceOptionsFromFlags(values sliceFlagValues, side string) (obj.SliceOptions, error) {
	options, err := obj.ParseSliceSide(side)
	if err != nil {
		return obj.SliceOptions{}, err
	}
	options.Boundary = *values.boundary
	options.Epsilon = *values.epsilon
	return options, nil
}

func newTransformFlagSet(name string) (*flag.FlagSet, transformFlagValues) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	values := transformFlagValues{
		uniformScale:   flags.Float64("scale", 1, "uniform scale factor"),
		scaleX:         flags.Float64("sx", 1, "x scale factor"),
		scaleY:         flags.Float64("sy", 1, "y scale factor"),
		scaleZ:         flags.Float64("sz", 1, "z scale factor"),
		translate:      flags.String("translate", "0,0,0", "translation vector x,y,z"),
		translateX:     flags.Float64("tx", 0, "x translation"),
		translateY:     flags.Float64("ty", 0, "y translation"),
		translateZ:     flags.Float64("tz", 0, "z translation"),
		rotateX:        flags.Float64("rx", 0, "x-axis rotation in degrees"),
		rotateY:        flags.Float64("ry", 0, "y-axis rotation in degrees"),
		rotateZ:        flags.Float64("rz", 0, "z-axis rotation in degrees"),
		matrixValue:    flags.String("matrix", "", "row-major affine 4x4 matrix"),
		center:         flags.Bool("center", false, "move bounding-box center to origin before other transforms"),
		normalize:      flags.Float64("normalize", 0, "center and scale max dimension to this size"),
		flipX:          flags.Bool("flip-x", false, "mirror across the X axis"),
		flipY:          flags.Bool("flip-y", false, "mirror across the Y axis"),
		flipZ:          flags.Bool("flip-z", false, "mirror across the Z axis"),
		reverseWinding: flags.Bool("reverse-winding", false, "reverse face winding after determinant handling"),
	}
	return flags, values
}

func transformOptionsFromFlags(values transformFlagValues) (obj.TransformOptions, error) {
	translation, err := parseVec3Flag(*values.translate)
	if err != nil {
		return obj.TransformOptions{}, fmt.Errorf("--translate: %w", err)
	}
	translation = translation.Add(obj.Vec3{X: *values.translateX, Y: *values.translateY, Z: *values.translateZ})

	var matrix *obj.Mat4
	if *values.matrixValue != "" {
		parsed, err := parseMat4Flag(*values.matrixValue)
		if err != nil {
			return obj.TransformOptions{}, fmt.Errorf("--matrix: %w", err)
		}
		matrix = &parsed
	}

	options := obj.DefaultTransformOptions()
	options.Center = *values.center
	options.NormalizeTo = *values.normalize
	options.Scale = obj.Vec3{
		X: *values.uniformScale * *values.scaleX,
		Y: *values.uniformScale * *values.scaleY,
		Z: *values.uniformScale * *values.scaleZ,
	}
	options.Translate = translation
	options.RotateDegrees = obj.Vec3{X: *values.rotateX, Y: *values.rotateY, Z: *values.rotateZ}
	options.Matrix = matrix
	options.FlipX = *values.flipX
	options.FlipY = *values.flipY
	options.FlipZ = *values.flipZ
	options.ReverseWinding = *values.reverseWinding
	return options, nil
}

func parseVec3Flag(value string) (obj.Vec3, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 3 {
		return obj.Vec3{}, errors.New("expected x,y,z")
	}

	x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return obj.Vec3{}, err
	}
	y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return obj.Vec3{}, err
	}
	z, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return obj.Vec3{}, err
	}
	return obj.Vec3{X: x, Y: y, Z: z}, nil
}

func parseMat4Flag(value string) (obj.Mat4, error) {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '[' || r == ']' || r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(parts) != 16 {
		return obj.Mat4{}, fmt.Errorf("expected 16 values, got %d", len(parts))
	}

	var matrix obj.Mat4
	for index, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return obj.Mat4{}, err
		}
		matrix[index/4][index%4] = value
	}
	return matrix, nil
}

func printInfo(writer io.Writer, path string, stats obj.Stats) {
	fmt.Fprintf(writer, "File: %s\n", path)
	fmt.Fprintf(writer, "Records: %d\n", stats.Records)
	fmt.Fprintf(writer, "Vertices: %d\n", stats.Vertices)
	fmt.Fprintf(writer, "Texture coords: %d\n", stats.TexCoords)
	fmt.Fprintf(writer, "Normals: %d\n", stats.Normals)
	fmt.Fprintf(writer, "Faces: %d\n", stats.Faces)
	fmt.Fprintf(writer, "Triangles: %d\n", stats.Triangles)
	fmt.Fprintf(writer, "Quads: %d\n", stats.Quads)
	fmt.Fprintf(writer, "N-gons: %d\n", stats.NGons)
	if stats.Faces > 0 {
		fmt.Fprintf(writer, "Face vertices: %d..%d\n", stats.FaceVertsMin, stats.FaceVertsMax)
	}
	if !stats.Bounds.Empty {
		size := stats.Bounds.Size()
		center := stats.Bounds.Center()
		fmt.Fprintf(writer, "Bounds min: %s %s %s\n", formatFloat(stats.Bounds.Min.X), formatFloat(stats.Bounds.Min.Y), formatFloat(stats.Bounds.Min.Z))
		fmt.Fprintf(writer, "Bounds max: %s %s %s\n", formatFloat(stats.Bounds.Max.X), formatFloat(stats.Bounds.Max.Y), formatFloat(stats.Bounds.Max.Z))
		fmt.Fprintf(writer, "Bounds size: %s %s %s\n", formatFloat(size.X), formatFloat(size.Y), formatFloat(size.Z))
		fmt.Fprintf(writer, "Bounds center: %s %s %s\n", formatFloat(center.X), formatFloat(center.Y), formatFloat(center.Z))
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, `meshtool manipulates Wavefront OBJ files.

Usage:
  meshtool info <input.obj>
  meshtool transform [options] <input.obj> <output.obj>
  meshtool triangulate <input.obj> <output.obj>
  meshtool slice [options] <x-|x+|y-|y+|z-|z+> <input.obj> <output.obj>
  meshtool chain <input.obj> <output.obj> <operation> [operation ...]

Chain operations:
  transform [options]
  triangulate
  slice [options] <x-|x+|y-|y+|z-|z+>

Slice options:
  -at <value>          plane coordinate; default 0
  -eps <value>         classification epsilon; default 1e-9

Transform options:
  -center              move bounding-box center to origin before other transforms
  -normalize <size>    center and scale max dimension to size
  -scale <factor>      uniform scale
  -sx/-sy/-sz <factor> per-axis scale
  -translate x,y,z     translation vector
  -tx/-ty/-tz <value>  per-axis translation added to -translate
  -rx/-ry/-rz <deg>    rotations in degrees, applied X then Y then Z
  -matrix <16 values>  row-major affine 4x4 applied after other transforms;
                       spaces, commas, and [] are accepted as separators
  -flip-x/-flip-y/-flip-z
                       mirror an axis and preserve outward winding
  -reverse-winding     reverse face winding after transform determinant handling

Use "-" as input or output for stdin/stdout.`)
}
