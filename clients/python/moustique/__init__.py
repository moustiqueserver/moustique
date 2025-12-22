from .client import Moustique
from .client import (
    getversion, getstats, getclients, gettopics,
    getposters, getpeerhosts, getcrooks
)

__version__ = "0.1.0"  # sätt en version
__all__ = [
    "Moustique",
    "getversion", "getstats", "getclients", "gettopics",
    "getposters", "getpeerhosts", "getcrooks"
]