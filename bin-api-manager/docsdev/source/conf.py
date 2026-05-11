# -*- coding: utf-8 -*-
#
# Configuration file for the Sphinx documentation builder.
#
# For the full list of built-in configuration values, see the documentation:
# https://www.sphinx-doc.org/en/master/usage/configuration.html

# -- Project information -----------------------------------------------------

project = "voipbin"
copyright = "2026, VoIPBIN"
author = "VoIPBIN"

# The short X.Y version
version = ""
# The full version, including alpha/beta/rc tags
release = ""


# -- General configuration ---------------------------------------------------

extensions = [
    "sphinxcontrib.youtube",
]

# Add any paths that contain templates here, relative to this directory.
templates_path = ["_templates"]

source_suffix = ".rst"
master_doc = "index"
language = "en"

# Files included via `.. include::` from a parent shim. They are NOT standalone
# documents and re-declare the same labels as the parent (which causes duplicate
# label warnings if Sphinx tries to build them as top-level pages). Excluding
# them from the build keeps the single-page UX of the parent shim while
# silencing the duplicate-label noise.
exclude_patterns = [
    "direct_hash_overview.rst",
    "intro_applications.rst",
    "intro_channels.rst",
    "quickstart_authentication.rst",
    "quickstart_call.rst",
    "quickstart_events.rst",
    "quickstart_extension.rst",
    "quickstart_realtime.rst",
    "quickstart_sandbox.rst",
    "quickstart_signup.rst",
    "sdk_overview.rst",
]


# -- Options for HTML output -------------------------------------------------

html_theme = "furo"

html_title = "VoIPBin Documentation"
html_short_title = "VoIPBin"
html_logo = "_static/images/voipbin-high-resolution-logo-white-transparent.png"

html_static_path = ["_static"]

# Pygments styles: separate light / dark so code blocks read well in both modes.
pygments_style = "tango"
pygments_dark_style = "monokai"

# Furo theme options. Top-level project links go to the navigation bar via
# `top_of_page_buttons` and the footer; the announcement bar surfaces the
# opensource positioning.
html_theme_options = {
    "announcement": (
        "VoIPBin is an opensource CPaaS platform. "
        '<a style="color: var(--color-announcement-text);" '
        'href="https://github.com/voipbin/voipbin">Star us on GitHub</a>'
    ),
    "sidebar_hide_name": False,
    "navigation_with_keys": True,
    "top_of_page_buttons": ["view", "edit"],
    "source_repository": "https://github.com/voipbin/monorepo",
    "source_branch": "main",
    "source_directory": "bin-api-manager/docsdev/source/",
    "footer_icons": [
        {
            "name": "GitHub",
            "url": "https://github.com/voipbin/voipbin",
            "html": "",
            "class": "fa-brands fa-solid fa-github fa-2x",
        },
    ],
    "light_css_variables": {
        "color-brand-primary": "#0066cc",
        "color-brand-content": "#0066cc",
        "color-announcement-background": "#0a2540",
        "color-announcement-text": "#ffffff",
    },
    "dark_css_variables": {
        "color-brand-primary": "#4da3ff",
        "color-brand-content": "#4da3ff",
        "color-announcement-background": "#0a2540",
        "color-announcement-text": "#ffffff",
    },
}


# -- Options for HTMLHelp output ---------------------------------------------

htmlhelp_basename = "voipbindoc"


# -- Options for LaTeX output ------------------------------------------------

latex_elements: dict = {}

latex_documents = [
    (master_doc, "voipbin.tex", "voipbin Documentation",
     "Sungtae Kim", "manual"),
]


# -- Options for manual page output ------------------------------------------

man_pages = [
    (master_doc, "voipbin", "voipbin Documentation",
     [author], 1)
]


# -- Options for Texinfo output ----------------------------------------------

texinfo_documents = [
    (master_doc, "voipbin", "voipbin Documentation",
     author, "voipbin", "Opensource CPaaS platform documentation.",
     "Miscellaneous"),
]


# -- Options for Epub output -------------------------------------------------

epub_title = project
epub_exclude_files = ["search.html"]
