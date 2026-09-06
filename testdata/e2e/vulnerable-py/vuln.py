"""Synthetic vulnerable fixture: must BLOCK the gate via Semgrep."""
import subprocess


def run_user(cmd):
    return subprocess.call(cmd, shell=True)
