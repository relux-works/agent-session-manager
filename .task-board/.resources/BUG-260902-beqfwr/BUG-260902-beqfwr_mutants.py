import subprocess, sys, pathlib, re
src = pathlib.Path("internal/config/sshargs.go")
original = src.read_text()

MUTANTS = [
 ("M1 permit -F flag",
  "\t'C': {}, // request compression",
  "\t'C': {}, // request compression\n\t'F': {}, // MUTANT: permit alternate config file"),
 ("M2 declare ProxyCommand permitted",
  '\t"user":                {permits: sshWordValue},',
  '\t"user":                {permits: sshWordValue},\n\t"proxycommand":        {permits: sshWordValue}, // MUTANT'),
 ("M3 widen StrictHostKeyChecking to the disabling aliases",
  '{permits: sshEnumeratedValue("yes"), hostAuthentication: true}',
  '{permits: sshEnumeratedValue("yes", "no", "off", "false", "accept-new"), hostAuthentication: true}'),
 ("M4 stop walking grouped short flags past the first letter",
  "\t\tgroup := argument[1:]",
  "\t\tgroup := argument[1:2] // MUTANT"),
 ("M5 declare Include permitted",
  '\t"user":                {permits: sshWordValue},',
  '\t"user":                {permits: sshWordValue},\n\t"include":             {permits: sshWordValue}, // MUTANT'),
 ("M6 declare KnownHostsCommand permitted",
  '\t"knownhostscommand":                {hostAuthentication: true},',
  '\t"knownhostscommand":                {permits: sshWordValue}, // MUTANT'),
 ("M7 declare PermitLocalCommand and LocalCommand permitted",
  '\t"user":                {permits: sshWordValue},',
  '\t"user":                {permits: sshWordValue},\n\t"permitlocalcommand":  {permits: sshWordValue}, // MUTANT\n\t"localcommand":        {permits: sshWordValue}, // MUTANT'),
 ("M8 admit bare operands instead of refusing them",
  "\t\tif len(argument) < 2 || argument[0] != '-' || argument == \"--\" {\n\t\t\treturn sshRefusalUnpermittedArgument\n\t\t}",
  "\t\tif len(argument) < 2 || argument[0] != '-' || argument == \"--\" {\n\t\t\tcontinue // MUTANT\n\t\t}"),
 ("M9 admit option names the registry does not declare",
  "\tif !declared {\n\t\treturn sshRefusalUnpermittedOption\n\t}",
  "\tif !declared {\n\t\treturn sshArgumentAdmitted // MUTANT\n\t}"),
 ("M10 let UserKnownHostsFile permit every value",
  '\t"userknownhostsfile":               {hostAuthentication: true},',
  '\t"userknownhostsfile":               {permits: func(string) bool { return true }, hostAuthentication: true},'),
]

log = []
try:
    for name, old, new in MUTANTS:
        assert original.count(old) == 1, f"{name}: anchor count {original.count(old)}"
        src.write_text(original.replace(old, new, 1))
        proc = subprocess.run(["go", "test", "./internal/config", "-count=1"],
                              capture_output=True, text=True)
        fails = sorted(set(re.findall(r"^\s*--- FAIL: (\S+)", proc.stdout, re.M)))
        top = [f for f in fails if "/" not in f]
        log.append((name, proc.returncode, top, len(fails)))
        print(f"{name}\n  exit={proc.returncode} failing_top_level={top} failing_nodes={len(fails)}\n")
finally:
    src.write_text(original)

bad = [l for l in log if l[1] == 0]
print("SUMMARY: %d/%d mutants reddened the suite" % (len(log)-len(bad), len(log)))
if bad:
    print("SURVIVING MUTANTS:", [b[0] for b in bad])
    sys.exit(1)
