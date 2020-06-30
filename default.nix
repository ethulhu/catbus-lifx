# SPDX-FileCopyrightText: 2020 Ethel Morgan
#
# SPDX-License-Identifier: MIT

{ pkgs ? import <nixpkgs> {} }:
with pkgs;

buildGoModule rec {
  name = "catbus-lifx-${version}";
  version = "latest";
  goPackagePath = "go.eth.moe/catbus-lifx";

  modSha256 = "0ixrjc9m2g2129n1z73ybhiiz7b40fsa7w24w5bnspd3x4wsyr7z";

  src = ./.;

  meta = {
    homepage = "https://ethulhu.co.uk/catbus";
    licence = stdenv.lib.licenses.mit;
  };
}
