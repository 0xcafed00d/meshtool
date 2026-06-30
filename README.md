# meshtool

`meshtool` is a Go command-line tool for inspecting and editing Wavefront OBJ files.

## Build

```sh
go build -o meshtool ./cmd/meshtool
```

## Commands

```sh
./meshtool info Mesh\ 0623_3.obj
./meshtool edges Mesh\ 0623_3.obj
./meshtool transform -normalize 1 Mesh\ 0623_3.obj normalized.obj
./meshtool transform -center -scale 0.5 -rz 90 Mesh\ 0623_3.obj rotated.obj
./meshtool transform -matrix "1 0 0 10 0 1 0 0 0 0 1 0 0 0 0 1" input.obj moved.obj
./meshtool transform -matrix "[[1,0,0,10], [0,1,0,0], [0,0,1,0], [0,0,0,1]]" input.obj moved.obj
./meshtool triangulate input.obj triangles.obj
./meshtool remesh -target 1.0 -iterations 5 input.obj remeshed.obj
./meshtool slice x+ input.obj positive-x.obj
./meshtool slice -at 12.5 z- input.obj below-z.obj
./meshtool chain input.obj output.obj slice x+ transform -normalize 1 triangulate transform -rz 90
```

`edges` reports unique triangle edge-length statistics: min, mean, median, p90, p95, and max. Non-triangle faces are measured after fan triangulation.

`remesh` rewrites the OBJ as plain vertices and triangle faces, splitting triangle edges longer than `target * max-factor` consistently across every face sharing that edge. This first-pass remesher improves oversized triangles in sparse scan regions, but it does not yet collapse dense regions, smooth vertices, project back onto the original surface, or preserve texture coordinates and normals.

`transform` supports centering, normalizing, uniform/per-axis scaling, translation, X/Y/Z rotations, row-major affine 4x4 matrices, axis flips, and winding reversal. Matrix transforms are applied after the other transform options. The parser preserves record order and passes through unsupported OBJ records unchanged.

`slice` clips faces against an axis-aligned plane. `x+`, `y+`, and `z+` keep coordinates greater than or equal to the plane; `x-`, `y-`, and `z-` keep coordinates less than or equal to it. The default plane coordinate is `0`; use `-at` to move it. Cut-edge vertices are welded across adjacent faces so the sliced boundary remains connected.

`chain` runs `transform`, `slice`, and `triangulate` operations in the order listed, without writing intermediate OBJ files.
