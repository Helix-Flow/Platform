from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="helixflow",
    version="1.0.0",
    author="HelixFlow Team",
    author_email="team@helixflow.ai",
    description="Official Python SDK for HelixFlow AI inference platform",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/helixflow/helixflow-python",
    packages=find_packages(),
    classifiers=[
        "Development Status :: 5 - Production/Stable",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
    ],
    python_requires=">=3.8",
    install_requires=[
        "requests>=2.25.0",
    ],
    extras_require={
        "cognee": ["neo4j>=5.0.0", "qdrant-client>=1.0.0"],
    },
)
