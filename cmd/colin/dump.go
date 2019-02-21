package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/evanphx/columbia/wasm"
	"github.com/evanphx/columbia/wasm/disasm"
)

func dump(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	mod, err := wasm.ReadModule(f, nil)
	if err != nil {
		return err
	}

	tr := tabwriter.NewWriter(os.Stdout, 4, 8, 1, ' ', 0)

	fmt.Printf("\n[sections]\n")
	for _, sec := range mod.Sections {
		tr.Write([]byte(sec.Description()))
	}

	tr.Flush()

	fmt.Printf("\n[imports]\n")
	for i, ii := range mod.Import.Entries {
		fmt.Printf("%3d %8s %s.%s\n", i, ii.Type.Kind(), ii.ModuleName, ii.FieldName)
	}

	fmt.Printf("\n[functions]\n")
	tr = tabwriter.NewWriter(os.Stdout, 4, 8, 1, ' ', 0)
	for i, fn := range mod.FunctionIndexSpace {
		var name string
		if fn.Body != nil {
			name = fn.Body.Name
			fmt.Fprintf(tr, "%d\t%s\toffset=%d len=%d\n", i, name,
				fn.Body.StartLoc, len(fn.Body.Code))
		} else if fn.ImportStub != nil {
			name = fn.ImportStub.Name
			fmt.Fprintf(tr, "%d\t%s\timport\n", i, name)
		}
	}

	tr.Flush()

	/*
		fmt.Printf("\n[code]\n")
		for i, b := range mod.Code.Bodies {
			fmt.Printf("%3d `%s` code_len=%d\n", i, b.Name, len(b.Code))
		}
	*/

	byOffset := make(map[uint32]int)

	fmt.Printf("\n[code relocations]\n")
	for i, reloc := range mod.CodeRelocations {
		fmt.Printf("%03x %5s +%d\n", reloc.Offset,
			reloc.StringType(), reloc.Addend)
		byOffset[reloc.Offset] = i
	}

	fmt.Printf("\n[data relocations]\n")
	for _, reloc := range mod.DataRelocations {
		fmt.Printf("%03x %5s +%d\n", reloc.Offset,
			reloc.StringType(), reloc.Addend)
	}

	for _, fn := range mod.FunctionIndexSpace {
		// Skip native methods as they need not be
		// disassembled; simply add them at the end
		// of the `funcs` array as is, as specified
		// in the spec. See the "host functions"
		// section of:
		// https://webassembly.github.io/spec/core/exec/modules.html#allocation
		if fn.IsHost() {
			continue
		}

		if fn.Body == nil {
			continue
		}

		code, err := disasm.Disassemble(fn.Body.Code, fn.Body.Loc)
		if err != nil {
			return err
		}

		totalLocalVars := 0
		totalLocalVars += len(fn.Sig.ParamTypes)
		for _, entry := range fn.Body.Locals {
			totalLocalVars += int(entry.Count)
		}

		fmt.Printf("\n%03x <%s>:\n", fn.Body.StartLoc, fn.Body.Name)

		tr = tabwriter.NewWriter(os.Stdout, 4, 8, 1, ' ', 0)

		for i, instr := range code {
			if len(instr.Immediates) == 0 {
				fmt.Fprintf(tr, "  %x\t%x\t%s\n", instr.Offset, instr.Op.Code, instr.Op.Name)
			} else {
				switch instr.Op.Code {
				case 0x41:
					if relocIdx, ok := byOffset[uint32(instr.Offset+1)]; ok {
						reloc := mod.CodeRelocations[relocIdx]
						fmt.Fprintf(tr, "  %x\t%x\t%s\t%v\t# %s reloc\n",
							instr.Offset, instr.Op.Code, instr.Op.Name,
							instr.Immediates[0], reloc.StringType())
						continue
					}
				case 0x28:
					if relocIdx, ok := byOffset[uint32(instr.Offset+2)]; ok {
						reloc := mod.CodeRelocations[relocIdx]
						fmt.Fprintf(tr, "  %x\t%x\t%s\t%v\t%v\t# %s reloc\n",
							instr.Offset, instr.Op.Code, instr.Op.Name,
							instr.Immediates[0], instr.Immediates[1],
							reloc.StringType())
						continue
					}
				}
				fmt.Fprintf(tr, "  %x\t%x\t%s\t", instr.Offset, instr.Op.Code, instr.Op.Name)

				for _, arg := range instr.Immediates {
					fmt.Fprintf(tr, "%v\t", arg)
				}

				switch instr.Op.Code {
				case 0x10:
					callee := mod.FunctionIndexSpace[int(instr.Immediates[0].(uint32))]
					fmt.Fprintf(tr, "# %s", callee.Name())
				default:
					if i != len(code)-1 {
						for j := instr.Offset; j < code[i+1].Offset; j++ {
							if relocIdx, ok := byOffset[uint32(j)]; ok {
								reloc := mod.CodeRelocations[relocIdx]
								fmt.Fprintf(tr, "# %s reloc", reloc.StringType())
							}
						}
					}
				}

				fmt.Fprintf(tr, "\n")
			}
		}

		tr.Flush()

		/*
			code, table, offsets := compile.Compile(disassembly.Code)
			pm.funcs[i] = &compiledFunction{
				name:           fn.Body.Name,
				code:           code,
				branchTables:   table,
				maxDepth:       disassembly.MaxDepth,
				totalLocalVars: totalLocalVars,
				args:           len(fn.Sig.ParamTypes),
				returns:        len(fn.Sig.ReturnTypes) != 0,
				offsets:        offsets,
			}
		*/
	}

	return nil
}
