# SPDX-FileCopyrightText: 2020 Ethel Morgan
#
# SPDX-License-Identifier: MIT

{ pkgs ? import <nixpkgs> {} }:
with pkgs;

buildGoModule rec {
  name = "catbus-lifx-${version}";
  version = "latest";
  goPackagePath = "go.eth.moe/catbus-lifx";

  modSha256 = "0jgc6p5y11ydwpk32hly7yv9w340scncg0gwvfb44vlrb9msccba";

  src = ./.;

  meta = {
    homepage = "https://ethulhu.co.uk/catbus";
    licence = stdenv.lib.licenses.mit;
  };
}
