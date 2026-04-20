"""Shared fixtures for skill validation-script tests."""

import os
import pytest


@pytest.fixture(scope="session")
def project_root():
    """Return the absolute path to the repository root."""
    return os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))


@pytest.fixture(scope="session")
def skills_root(project_root):
    """Return the absolute path to .agents/skills/."""
    return os.path.join(project_root, ".agents", "skills")
