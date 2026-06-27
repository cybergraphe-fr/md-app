#!/usr/bin/env python3
"""SSRF/LFI-hardened WeasyPrint CLI wrapper.

Usage: weasyprint_safe.py INPUT.html OUTPUT.pdf

The PDF export pipeline feeds WeasyPrint HTML derived from 100% user-controlled
Markdown. WeasyPrint will, by default, fetch every referenced resource. That
turns the export endpoint into a Server-Side Request Forgery primitive
(e.g. ``![](http://169.254.169.254/latest/meta-data/...)`` would let an
unauthenticated caller read cloud-metadata / internal services and exfiltrate
the response inside the produced PDF), as well as a Local File disclosure
primitive (``<img src="file:///etc/passwd">``).

This wrapper routes every fetch through ``safe_url_fetcher`` which:
  * allows ``data:`` URIs (inline images, e.g. rendered Mermaid PNGs);
  * allows ``file://`` only under an explicit allow-list of directories
    (the bundled print.css + fonts, the input HTML's own temp dir, system
    font dirs) — everything else (``/etc``, ``/run/secrets``, ``/data`` …)
    is rejected;
  * allows ``http(s)`` only when the host resolves exclusively to public
    IP addresses — private, loopback, link-local, reserved, multicast and
    cloud-metadata ranges are rejected;
  * rejects every other scheme.

The pandoc HTML stage no longer uses ``--embed-resources`` (which would make
*pandoc* the SSRF egress point instead), so this wrapper is the single,
guarded network/file egress for the whole PDF pipeline.
"""

import ipaddress
import os
import socket
import sys
from urllib.parse import urlsplit, unquote

from weasyprint import HTML

try:  # url_fetcher location moved across WeasyPrint versions
    from weasyprint.urls import default_url_fetcher
except ImportError:  # pragma: no cover - depends on installed version
    from weasyprint import default_url_fetcher


def _allowed_file_roots(html_path):
    roots = [
        "/app/pandoc",          # bundled print.css + fonts
        "/usr/share/fonts",
        "/usr/share/weasyprint",
        "/etc/fonts",
    ]
    try:
        roots.append(os.path.dirname(os.path.realpath(html_path)))
    except OSError:
        pass
    return [os.path.realpath(r) + os.sep for r in roots]


def _is_blocked_ip(ip_str):
    try:
        ip = ipaddress.ip_address(ip_str)
    except ValueError:
        return True
    if (
        ip.is_private
        or ip.is_loopback
        or ip.is_link_local
        or ip.is_reserved
        or ip.is_multicast
        or ip.is_unspecified
    ):
        return True
    # Explicit cloud metadata endpoints (IMDS) — belt and suspenders.
    if str(ip) in ("169.254.169.254", "fd00:ec2::254"):
        return True
    # IPv4-mapped / 6to4 / Teredo wrappers around private space.
    if isinstance(ip, ipaddress.IPv6Address) and ip.ipv4_mapped is not None:
        return _is_blocked_ip(str(ip.ipv4_mapped))
    return False


def _make_fetcher(allowed_roots):
    def safe_url_fetcher(url, *args, **kwargs):
        parts = urlsplit(url)
        scheme = parts.scheme.lower()

        if scheme in ("data", ""):
            return default_url_fetcher(url, *args, **kwargs)

        if scheme == "file":
            target = os.path.realpath(unquote(parts.path))
            for root in allowed_roots:
                if (target + os.sep).startswith(root) or target == root.rstrip(os.sep):
                    return default_url_fetcher(url, *args, **kwargs)
            raise ValueError("blocked local file outside allow-list: %s" % target)

        if scheme not in ("http", "https"):
            raise ValueError("blocked URL scheme: %s" % scheme)

        host = parts.hostname
        if not host:
            raise ValueError("blocked URL: missing host")
        try:
            infos = socket.getaddrinfo(host, parts.port or None)
        except socket.gaierror as exc:
            raise ValueError("blocked URL: DNS resolution failed: %s" % host) from exc
        if not infos:
            raise ValueError("blocked URL: no address for host: %s" % host)
        for info in infos:
            ip_str = info[4][0]
            if _is_blocked_ip(ip_str):
                raise ValueError(
                    "blocked SSRF target: %s -> %s" % (host, ip_str)
                )
        return default_url_fetcher(url, *args, **kwargs)

    return safe_url_fetcher


def main(argv):
    if len(argv) != 3:
        sys.stderr.write("usage: weasyprint_safe.py INPUT.html OUTPUT.pdf\n")
        return 2
    html_path, pdf_path = argv[1], argv[2]
    fetcher = _make_fetcher(_allowed_file_roots(html_path))
    HTML(filename=html_path, url_fetcher=fetcher).write_pdf(pdf_path)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
