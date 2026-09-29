"""Exercise installer behavior without network access or administrator privileges."""

import hashlib
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest


INSTALLER = Path(__file__).resolve().parents[1] / "install.sh"


class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="remove-shit-install-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / "commands"
        self.bin.mkdir()
        self.target = self.root / "install with spaces"
        self.env = dict(os.environ)
        self.env.update(
            PATH=str(self.bin) + os.pathsep + os.environ["PATH"],
            INSTALL_DIR=str(self.target),
            QA_ROOT=str(self.root),
            QA_OS="Darwin",
            QA_ARCH="arm64",
        )
        self.env.pop("REMOVE_SHIT_VERSION", None)
        self.command("uname", '#!/bin/sh\ncase "$1" in -s) echo "$QA_OS" ;; -m) echo "$QA_ARCH" ;; esac\n')
        self.command("sudo", "#!/bin/sh\necho 'Unexpected sudo call' >&2\nexit 99\n")
        self.command("curl", '''#!/usr/bin/env python3
import os, pathlib, shutil, sys
args = sys.argv[1:]
urls = [arg for arg in args if arg.startswith("https://")]
assert len(urls) == 1, args
url = urls[0]
if url.endswith("/releases/latest"):
    print("https://github.com/michaelmjhhhh/remove-shit/releases/tag/v0.1.0", end="")
else:
    assert "/releases/download/v0.1.0/" in url, url
    dest = args[args.index("-o") + 1]
    source = pathlib.Path(os.environ["QA_ROOT"]) / url.rsplit("/", 1)[1]
    shutil.copyfile(source, dest)
''')
        checksums = []
        for arch in ("arm64", "amd64"):
            archive = self.root / f"remove-shit_darwin_{arch}.tar.gz"
            content = b"#!/bin/sh\necho 'remove-shit v0.1.0'\n"
            with tarfile.open(archive, "w:gz") as tar:
                info = tarfile.TarInfo("remove-shit")
                info.size = len(content)
                info.mode = 0o755
                tar.addfile(info, io.BytesIO(content))
            checksums.append(f"{hashlib.sha256(archive.read_bytes()).hexdigest()}  {archive.name}\n")
        (self.root / "checksums.txt").write_text("".join(checksums))

    def command(self, name, content):
        path = self.bin / name
        path.write_text(content)
        path.chmod(0o755)

    def run_installer(self):
        # Pipe the script exactly as in the documented curl | sh installation.
        return subprocess.run(
            ["sh"], input=INSTALLER.read_text(), env=self.env,
            capture_output=True, text=True, timeout=15,
        )

    def test_latest_apple_silicon(self):
        result = self.run_installer()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        binary = self.target / "remove-shit"
        self.assertTrue(os.access(binary, os.X_OK))
        self.assertEqual(subprocess.check_output([str(binary)], text=True).strip(), "remove-shit v0.1.0")

    def test_pinned_intel(self):
        self.env.update(QA_ARCH="x86_64", REMOVE_SHIT_VERSION="v0.1.0")
        result = self.run_installer()
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("amd64", result.stdout)

    def test_corrupt_download_preserves_existing_install(self):
        self.target.mkdir()
        binary = self.target / "remove-shit"
        binary.write_text("original installation")
        with (self.root / "remove-shit_darwin_arm64.tar.gz").open("ab") as archive:
            archive.write(b"corrupted")
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Checksum verification failed", result.stderr)
        self.assertEqual(binary.read_text(), "original installation")

    def test_missing_checksum(self):
        (self.root / "checksums.txt").write_text("")
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("checksum is missing", result.stderr)
        self.assertFalse(self.target.exists())

    def test_unsupported_os(self):
        self.env["QA_OS"] = "Linux"
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Only macOS", result.stderr)

    def test_unsupported_architecture(self):
        self.env["QA_ARCH"] = "riscv64"
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Unsupported Mac architecture", result.stderr)

    def test_rejects_invalid_version_and_relative_destination(self):
        self.env["REMOVE_SHIT_VERSION"] = "v0.1.0/../../bad"
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.env["REMOVE_SHIT_VERSION"] = "v0.1.0"
        self.env["INSTALL_DIR"] = "relative/path"
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("absolute path", result.stderr)


if __name__ == "__main__":
    unittest.main()
