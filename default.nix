# SPDX-FileCopyrightText: 2020 Ethel Morgan
#
# SPDX-License-Identifier: MIT

{ pkgs ? import <nixpkgs> {} }:
with pkgs;

buildGoModule rec {
  name = "catbus-lifx-${version}";
  version = "latest";
  goPackagePath = "go.eth.moe/catbus-lifx";

  modSha256 = "1hrx6m8bcmw43vcm0h9s0pinsb0gxwzv6w5jh3rvs83mkl4m7fii";

  src = ./.;

  meta = {
    homepage = "https://ethulhu.co.uk/catbus";
    licence = stdenv.lib.licenses.mit;
  };
}
