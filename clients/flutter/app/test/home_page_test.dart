/// Widget tests for HomePage using FakeCoreClient (no daemon needed).
library;

import 'package:core_client/src/fake.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:weftlink_app/home_page.dart';

Widget _wrap(FakeCoreClient fake) => MaterialApp(
      home: HomePage(
        factory: (host, port) => fake,
        initialHost: 'fake',
        initialPort: 1,
      ),
    );

void main() {
  testWidgets('shows connected status from core', (tester) async {
    final fake = FakeCoreClient();
    await tester.pumpWidget(_wrap(fake));
    await tester.pumpAndSettle();

    expect(find.text('Connected'), findsOneWidget);
    expect(find.text('fake-node'), findsOneWidget);
    expect(find.text('0.1.0'), findsOneWidget);
  });

  testWidgets('lists peers after knot', (tester) async {
    final fake = FakeCoreClient();
    await tester.pumpWidget(_wrap(fake));
    await tester.pumpAndSettle();

    expect(find.textContaining('No peers woven in'), findsOneWidget);

    await fake.knot('ABC123');
    await tester.pumpAndSettle();

    expect(find.text('fake-ABC123'), findsOneWidget);
    expect(find.textContaining('Peers (1)'), findsOneWidget);
  });

  testWidgets('ping shows latency', (tester) async {
    final fake = FakeCoreClient()..pingLatencyMs = 7;
    await tester.pumpWidget(_wrap(fake));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Ping'));
    await tester.pumpAndSettle();

    expect(find.text('Ping · 7ms'), findsOneWidget);
    expect(find.textContaining('ping 7ms'), findsOneWidget);
  });
}