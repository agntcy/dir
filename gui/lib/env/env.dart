// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'package:envied/envied.dart';

part 'env.g.dart';

@Envied(path: '.env')
abstract class Env {
  @EnviedField(varName: 'MEASUREMENT_ID', obfuscate: true, defaultValue: '')
  static final String measurementId = _Env.measurementId;

  @EnviedField(varName: 'API_SECRET', obfuscate: true, defaultValue: '')
  static final String apiSecret = _Env.apiSecret;
}
