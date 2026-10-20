# Draw the source and binary SBOMs as ASCII trees so their shapes can be compared.
#   $src: cyclonedx-gomod app  (from source)
#   $bin: cyclonedx-gomod bin  (from the built binary)

def nodename($idx; $ref):
  ($idx[$ref] // null) | if . then "\(.name)@\(.version)" else $ref end;

# Everything reachable from the main component, drawn as a tree.
def tree($b):
  ($b.metadata.component["bom-ref"]) as $main
  | ([$b.components[]? | {key: .["bom-ref"], value: .}] | from_entries) as $idx
  | ([$b.dependencies[]? | {key: .ref, value: (.dependsOn // [])}] | from_entries) as $deps
  | def lines($ref; $prefix; $path):
      ($deps[$ref] // []) as $kids
      | ($kids | length) as $n
      | range(0; $n) as $i
      | $kids[$i] as $c
      | ($i == $n - 1) as $last
      | ($path | index($c) != null) as $cycle
      | ($prefix + (if $last then "`-- " else "+-- " end) + nodename($idx; $c)
         + (if $cycle then " (cycle)" else "" end)),
        (if $cycle then empty
         else lines($c; $prefix + (if $last then "    " else "|   " end); $path + [$c]) end);
    ($b.metadata.component | "\(.name)@\(.version)  [main]"),
    lines($main; ""; [$main]);

# Components present in the SBOM that no edge connects to the main component.
def unlinked($b):
  ($b.metadata.component["bom-ref"]) as $main
  | ([$b.dependencies[]? | {key: .ref, value: (.dependsOn // [])}] | from_entries) as $deps
  | def reach($r; $seen):
      if ($seen | index($r)) then $seen
      else reduce ($deps[$r] // [])[] as $c ($seen + [$r]; reach($c; .)) end;
    reach($main; []) as $linked
  | [$b.components[]? | select(.["bom-ref"] as $r | $linked | index($r) | not)
     | "\(.name)@\(.version)" + (if .type == "platform" then "  [go toolchain]" else "" end)];

def section($title; $how; $b; $note):
  "=== \($title)",
  "    made by: \($how)",
  "",
  tree($b),
  (unlinked($b) as $u
   | if ($u | length) > 0
     then "", "    components with no relationship to the main module:", ($u[] | "      - " + .)
     else empty end),
  "",
  "    \($note)",
  "";

def names($b): [$b.components[]? | select(.type != "platform") | .name] | sort;
def list: if length == 0 then "none" else join(", ") end;

section("SOURCE  (\($srcfile))"; "cyclonedx-gomod app, reads go.mod and the package imports"; $src[0];
        "Real dependency graph: shows which module pulls in which (note x/sync under three parents)."),
section("BINARY  (\($binfile))"; "cyclonedx-gomod bin, reads the build info embedded in bin/guestbook"; $bin[0];
        "Build info lists modules but not who imports whom, so every module hangs directly off main."),
"=== Module differences",
"    only in binary vs source:  \((names($bin[0]) - names($src[0])) | list)",
"    only in source vs binary:  \((names($src[0]) - names($bin[0])) | list)"
