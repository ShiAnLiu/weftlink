/// Widget tests for HomePage using FakeCoreClient (no daemon needed).
library;

import 'package:core_client/src/fake.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:weftlink_app/home_page.dart';

void main() {
  testWidgets('shows connected status from core', (tester) async {
    final fake = FakeCoreClient();
    await tester.pumpWidget(MaterialApp(home: HomePage(client: fake)));
    await tester.pumpAndSettle();

    expect(find.text('Connected'), findsOneWidget);
    expect(find.text('fake-node'), findsOneWidget);
    expect(find.text('0.1.0'), findsOneWidget);
  });

  testWidgets('lists peers after knot', (tester) async {
    final fake = FakeCoreClient();
    await tester.pumpWidget(MaterialApp(home: HomePage(client: fake)));
    await tester.pumpAndSettle();

    expect(find.textContaining('No peers woven in'), findsOneWidget);

    await fake.knot('ABC123');
    await tester.pumpAndSettle();

    expect(find.text('fake-ABC123'), findsOneWidget);
    expect(find.textContaining('Peers (1)'), findsOneWidget);
  });
}