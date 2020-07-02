# SPDX-FileCopyrightText: 2020 Ethel Morgan
#
# SPDX-License-Identifier: MIT

{ pkgs ? import <nixpkgs> {} }:
with pkgs;

buildGoModule rec {
  name = "catbus-lifx-${version}";
  version = "latest";
  goPackagePath = "go.eth.moe/catbus-lifx";

  modSha256 = "19hyb5h0qxfjkmissxvhcrr7xdd66iss6v1w0nmz8zag7q1qk34r";

  src = ./.;

  meta = {
    homepage = "https://ethulhu.co.uk/catbus";
    licence = stdenv.lib.licenses.mit;
  };
}
