import { expect, test } from "bun:test";
import { errorText } from "./errortext";

test("the message of an Error, a string or an object with a message is shown as it is", () => {
  expect(errorText(new Error("external id \"WP1\" is already used"))).toBe('external id "WP1" is already used');
  expect(errorText("plain text")).toBe("plain text");
  expect(errorText({ message: "from the backend", kind: "RuntimeError" })).toBe("from the backend");
});

test("anything else is turned into text and never throws", () => {
  expect(errorText(undefined)).toBe("undefined");
  expect(errorText(42)).toBe("42");
  expect(errorText(null)).toBe("null");
  // an empty message must not make the explanation disappear
  expect(errorText(new Error(""))).toBe("The change was refused.");
  expect(errorText("   ")).toBe("The change was refused.");
  expect(errorText({ message: "" })).toBe("The change was refused.");
});
