// Copyright (c) 2026 dotandev
// SPDX-License-Identifier: MIT OR Apache-2.0

import {
  normalizeTokenLabel,
  resolvePkcs11KeyIdHex,
  resolveYkcs11KeyIdHex,
  Pkcs11Signer,
} from "../src/audit/signing/pkcs11Signer";

jest.mock("pkcs11js");

type Pkcs11MockModule = {
  __resetState: () => void;
  __setFailure: (
    method: string,
    failure: {
      message: string;
      code?: number;
      method?: string;
      times?: number;
    },
  ) => void;
  __setSlots: (slots: number[]) => void;
  __setTokenLabels: (labels: Record<string, string>) => void;
  __setKeys: (keys: number[]) => void;
};

const loadSigner = (): typeof Pkcs11Signer =>
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  require("../src/audit/signing/pkcs11Signer").Pkcs11Signer;

const pkcs11Mock = (): Pkcs11MockModule =>
  jest.requireMock("pkcs11js") as Pkcs11MockModule;

const setRequiredPkcs11Env = (): void => {
  process.env.GLASSBOX_PKCS11_MODULE = "/usr/lib/softhsm/libsofthsm2.so";
  process.env.GLASSBOX_PKCS11_PIN = "1234";
  process.env.GLASSBOX_PKCS11_KEY_LABEL = "test-key";
  delete process.env.GLASSBOX_PKCS11_KEY_ID;
  delete process.env.GLASSBOX_PKCS11_PIV_SLOT;
  delete process.env.GLASSBOX_PKCS11_SLOT;
  delete process.env.GLASSBOX_PKCS11_TOKEN_LABEL;
  delete process.env.GLASSBOX_PKCS11_PUBLIC_KEY_PEM;
  delete process.env.GLASSBOX_PKCS11_ALGORITHM;
};

describe("Pkcs11Signer", () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.resetModules();
    process.env = { ...originalEnv };
    pkcs11Mock().__resetState();
    setRequiredPkcs11Env();
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  describe("constructor validation", () => {
    test("throws clear error when GLASSBOX_PKCS11_MODULE is not set", () => {
      delete process.env.GLASSBOX_PKCS11_MODULE;

      const Signer = loadSigner();
      expect(() => new Signer()).toThrow(
        "pkcs11 provider selected but GLASSBOX_PKCS11_MODULE is not set",
      );
    });

    test("throws clear error when GLASSBOX_PKCS11_PIN is not set", () => {
      delete process.env.GLASSBOX_PKCS11_PIN;

      const Signer = loadSigner();
      expect(() => new Signer()).toThrow(
        "pkcs11 provider selected but GLASSBOX_PKCS11_PIN is not set",
      );
    });

    test("throws clear error when neither key label, key ID, nor PIV slot is set", () => {
      delete process.env.GLASSBOX_PKCS11_KEY_LABEL;
      delete process.env.GLASSBOX_PKCS11_KEY_ID;
      delete process.env.GLASSBOX_PKCS11_PIV_SLOT;

      const Signer = loadSigner();
      expect(() => new Signer()).toThrow(
        "pkcs11 provider selected but neither GLASSBOX_PKCS11_KEY_LABEL, GLASSBOX_PKCS11_KEY_ID, nor GLASSBOX_PKCS11_PIV_SLOT is set",
      );
    });

    test("rejects invalid explicit key IDs during construction", () => {
      process.env.GLASSBOX_PKCS11_KEY_ID = "xyz";
      delete process.env.GLASSBOX_PKCS11_KEY_LABEL;

      const Signer = loadSigner();
      expect(() => new Signer()).toThrow(
        "Invalid GLASSBOX_PKCS11_KEY_ID 'xyz'. Expected an even-length hex string (e.g., 01, 0a, 10).",
      );
    });

    test("rejects invalid slot index during construction", () => {
      process.env.GLASSBOX_PKCS11_SLOT = "abc";

      const Signer = loadSigner();
      expect(() => new Signer()).toThrow(
        "Invalid GLASSBOX_PKCS11_SLOT 'abc'. Expected a non-negative integer.",
      );
    });

    test("rejects unsupported algorithms during construction", () => {
      process.env.GLASSBOX_PKCS11_ALGORITHM = "rsa";

      const Signer = loadSigner();
      expect(() => new Signer()).toThrow(
        "Unsupported GLASSBOX_PKCS11_ALGORITHM 'rsa'. Supported values: ed25519, secp256k1.",
      );
    });

    test("rejects malformed public key PEM during construction", () => {
      process.env.GLASSBOX_PKCS11_PUBLIC_KEY_PEM = "not-a-pem";

      const Signer = loadSigner();
      expect(() => new Signer()).toThrow(
        "Invalid GLASSBOX_PKCS11_PUBLIC_KEY_PEM. Expected a SPKI PEM public key.",
      );
    });
  });

  describe("public_key", () => {
    test("returns public key from environment when set", async () => {
      process.env.GLASSBOX_PKCS11_PUBLIC_KEY_PEM =
        "-----BEGIN PUBLIC KEY-----\nMCowBQYDK2VwAyEA7k7O6Y8bD4b2M6h0x9YfM8Kq2jv4m7Y8Ww8U6F6K3qk=\n-----END PUBLIC KEY-----";

      const Signer = loadSigner();
      const signer = new Signer();
      const publicKey = await signer.public_key();

      expect(publicKey).toBe(process.env.GLASSBOX_PKCS11_PUBLIC_KEY_PEM);
    });

    test("throws clear error when public key is not configured", async () => {
      const Signer = loadSigner();
      const signer = new Signer();

      await expect(signer.public_key()).rejects.toThrow(
        "pkcs11 public key retrieval is not configured. Set GLASSBOX_PKCS11_PUBLIC_KEY_PEM to a SPKI PEM public key.",
      );
    });
  });

  describe("signing failures are actionable", () => {
    test("provides context for module load failures", async () => {
      pkcs11Mock().__setFailure("load", { message: "ENOENT" });

      const Signer = loadSigner();
      const signer = new Signer();

      await expect(signer.sign(Buffer.from("hello"))).rejects.toThrow(
        "Failed to load PKCS#11 module at '/usr/lib/softhsm/libsofthsm2.so': ENOENT. Check that the library exists and is accessible.",
      );
    });

    test("includes available tokens when token label selection fails", async () => {
      process.env.GLASSBOX_PKCS11_TOKEN_LABEL = "WantedToken";
      pkcs11Mock().__setTokenLabels({ "1": "OtherToken" });

      const Signer = loadSigner();
      const signer = new Signer();

      await expect(signer.sign(Buffer.from("hello"))).rejects.toThrow(
        "GLASSBOX_PKCS11_TOKEN_LABEL (WantedToken) did not match any tokens. Available tokens: OtherToken.",
      );
    });

    test("surfaces actionable login failures", async () => {
      pkcs11Mock().__setFailure("C_Login", {
        message: "Wrong PIN",
        code: 0x000000a0,
        method: "C_Login",
      });

      const Signer = loadSigner();
      const signer = new Signer();

      await expect(signer.sign(Buffer.from("hello"))).rejects.toThrow(
        "PKCS#11 login failed: C_Login: Wrong PIN (0x000000a0). Verify GLASSBOX_PKCS11_PIN and ensure the token is inserted and unlocked.",
      );
    });

    test("surfaces actionable key lookup failures when the key is missing", async () => {
      pkcs11Mock().__setKeys([]);

      const Signer = loadSigner();
      const signer = new Signer();

      await expect(signer.sign(Buffer.from("hello"))).rejects.toThrow(
        "Private key not found for label 'test-key'. Check the configured key selector and confirm the key exists on the token.",
      );
    });

    test("rejects out-of-range slot indexes with an explicit diagnostic", async () => {
      process.env.GLASSBOX_PKCS11_SLOT = "2";
      pkcs11Mock().__setSlots([1]);

      const Signer = loadSigner();
      const signer = new Signer();

      await expect(signer.sign(Buffer.from("hello"))).rejects.toThrow(
        "GLASSBOX_PKCS11_SLOT (2) is out of range. Available slot indexes: 0-0.",
      );
    });
  });

  describe("YubiKey PIV helpers", () => {
    test("normalizes token labels by trimming padding", () => {
      expect(normalizeTokenLabel("YubiKey PIV\x00\x00  ")).toBe("YubiKey PIV");
    });

    test("maps PIV slots to YKCS11 key IDs", () => {
      expect(resolveYkcs11KeyIdHex("9a")).toBe("01");
      expect(resolveYkcs11KeyIdHex("0x9c")).toBe("02");
      expect(resolveYkcs11KeyIdHex("9D")).toBe("03");
      expect(resolveYkcs11KeyIdHex("9e")).toBe("04");
      expect(resolveYkcs11KeyIdHex("82")).toBe("05");
      expect(resolveYkcs11KeyIdHex("95")).toBe("18");
      expect(resolveYkcs11KeyIdHex("f9")).toBe("19");
    });

    test("rejects unsupported PIV slots", () => {
      expect(() => resolveYkcs11KeyIdHex("9b")).toThrow(
        "Unsupported PIV slot '9b'. Supported slots: 9a, 9c, 9d, 9e, 82-95, f9.",
      );
    });

    test("prefers explicit key IDs over derived PIV slot IDs", () => {
      expect(resolvePkcs11KeyIdHex({ keyIdHex: "0a", pivSlot: "9a" })).toBe(
        "0a",
      );
    });

    test("rejects invalid explicit key IDs", () => {
      expect(() => resolvePkcs11KeyIdHex({ keyIdHex: "xyz" })).toThrow(
        "Invalid GLASSBOX_PKCS11_KEY_ID 'xyz'. Expected an even-length hex string (e.g., 01, 0a, 10).",
      );
      expect(() => resolvePkcs11KeyIdHex({ keyIdHex: "a" })).toThrow(
        "Invalid GLASSBOX_PKCS11_KEY_ID 'a'. Expected an even-length hex string (e.g., 01, 0a, 10).",
      );
    });
  });
});
