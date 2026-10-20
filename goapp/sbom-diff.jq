# Compare the runtime SBOM ($rt, flat) with the source SBOM ($src, graph).
def names: [.components[]? | select(.type != "platform") | .name] | sort;
def edges: [.dependencies[]? | (.dependsOn // []) | length] | add // 0;
def summary($label; $b):
  "\($label)",
  "  main component:      \($b.metadata.component.name)",
  "  components:          \($b.components | length)",
  "  go toolchain listed: \(if ([$b.components[]? | select(.type == "platform")] | length) > 0 then "yes" else "no" end)",
  "  dependency entries:  \(($b.dependencies // []) | length)",
  "  dependsOn edges:     \($b | edges)",
  "  dependency graph:    \(if ($b.dependencies // [] | length) > 0 then "present" else "absent (flat list)" end)";
def list: if length == 0 then "none" else join(", ") end;

summary("runtime (\($rtfile))"; $rt[0]),
"",
summary("source  (\($srcfile))"; $src[0]),
"",
"modules only in runtime SBOM: \((($rt[0] | names) - ($src[0] | names)) | list)",
"modules only in source SBOM:  \((($src[0] | names) - ($rt[0] | names)) | list)"
