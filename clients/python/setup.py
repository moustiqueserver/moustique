from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="moustique_client",
    version="0.1.1",
    author="Moustique",
    author_email="mulf@protonmail.com",
    description="Python client for Moustique messaging server",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/moustiqueserver/moustique",
    packages=find_packages(),
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Topic :: System :: Networking",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "License :: OSI Approved :: GNU General Public License v3 (GPLv3)",
        "Operating System :: OS Independent",
    ],
    python_requires='>=3.7',
    install_requires=[
        "requests>=2.25.0",
    ],
    keywords="moustique pubsub messaging realtime",
    project_urls={
        "Bug Reports": "https://github.com/moustiqueserver/moustique/issues",
        "Source": "https://github.com/moustiqueserver/moustique",
        "Documentation": "https://github.com/moustiqueserver/moustique#readme",
    },
)
