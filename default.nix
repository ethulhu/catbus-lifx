# SPDX-FileCopyrightText: 2020 Ethel Morgan
#
# SPDX-License-Identifier: MIT

{ pkgs ? import <nixpkgs> {} }:
with pkgs;

buildGoModule rec {
  name = "catbus-lifx-${version}";
  version = "latest";
  goPackagePath = "github.com/ethulhu/catbus-lifx";

  modSha256 = "1vc1sq6rhs2s1fa50vq4475qiixy2nd8npcsy76l4pkkxlibj25h";

  src = ./.;

  meta = {
    homepage = "https://ethulhu.co.uk/catbus";
    licence = stdenv.lib.licenses.mit;
  };
}
