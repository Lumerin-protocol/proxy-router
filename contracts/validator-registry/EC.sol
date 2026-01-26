//SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * Credits (multiple variations/forks)
 *
 * androlo: https://github.com/androlo/standard-contracts
 * witnet: https://github.com/witnet/elliptic-curve-solidity
 * jbaylina: https://github.com/jbaylina/ecsol
 * k06a: https://github.com/1Address/ecsol
 *
 */
contract EC {
    uint256 public constant gx = 0x79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798;
    uint256 public constant gy = 0x483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8;
    uint256 public constant n = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F;
    uint256 public constant a = 0;
    uint256 public constant b = 7;

    function _jAdd(uint256 x1, uint256 z1, uint256 x2, uint256 z2) public pure returns (uint256 x3, uint256 z3) {
        (x3, z3) = (addmod(mulmod(z2, x1, n), mulmod(x2, z1, n), n), mulmod(z1, z2, n));
    }

    function _jSub(uint256 x1, uint256 z1, uint256 x2, uint256 z2) public pure returns (uint256 x3, uint256 z3) {
        (x3, z3) = (addmod(mulmod(z2, x1, n), mulmod(n - x2, z1, n), n), mulmod(z1, z2, n));
    }

    function _jMul(uint256 x1, uint256 z1, uint256 x2, uint256 z2) public pure returns (uint256 x3, uint256 z3) {
        (x3, z3) = (mulmod(x1, x2, n), mulmod(z1, z2, n));
    }

    function _jDiv(uint256 x1, uint256 z1, uint256 x2, uint256 z2) public pure returns (uint256 x3, uint256 z3) {
        (x3, z3) = (mulmod(x1, z2, n), mulmod(z1, x2, n));
    }

    function _inverse(uint256 val) public pure returns (uint256 invVal) {
        uint256 t = 0;
        uint256 newT = 1;
        uint256 r = n;
        uint256 newR = val;
        uint256 q;
        while (newR != 0) {
            q = r / newR;

            (t, newT) = (newT, addmod(t, (n - mulmod(q, newT, n)), n));
            (r, newR) = (newR, r - q * newR);
        }

        return t;
    }

    function _ecAdd(uint256 x1, uint256 y1, uint256 z1, uint256 x2, uint256 y2, uint256 z2)
        public
        pure
        returns (uint256 x3, uint256 y3, uint256 z3)
    {
        uint256 lx;
        uint256 lz;
        uint256 da;
        uint256 db;

        if (x1 == 0 && y1 == 0) {
            return (x2, y2, z2);
        }

        if (x2 == 0 && y2 == 0) {
            return (x1, y1, z1);
        }

        if (x1 == x2 && y1 == y2) {
            (lx, lz) = _jMul(x1, z1, x1, z1);
            (lx, lz) = _jMul(lx, lz, 3, 1);
            (lx, lz) = _jAdd(lx, lz, a, 1);

            (da, db) = _jMul(y1, z1, 2, 1);
        } else {
            (lx, lz) = _jSub(y2, z2, y1, z1);
            (da, db) = _jSub(x2, z2, x1, z1);
        }

        (lx, lz) = _jDiv(lx, lz, da, db);

        (x3, da) = _jMul(lx, lz, lx, lz);
        (x3, da) = _jSub(x3, da, x1, z1);
        (x3, da) = _jSub(x3, da, x2, z2);

        (y3, db) = _jSub(x1, z1, x3, da);
        (y3, db) = _jMul(y3, db, lx, lz);
        (y3, db) = _jSub(y3, db, y1, z1);

        if (da != db) {
            x3 = mulmod(x3, db, n);
            y3 = mulmod(y3, da, n);
            z3 = mulmod(da, db, n);
        } else {
            z3 = da;
        }
    }

    function _ecDouble(uint256 x1, uint256 y1, uint256 z1) public pure returns (uint256 x3, uint256 y3, uint256 z3) {
        (x3, y3, z3) = _ecAdd(x1, y1, z1, x1, y1, z1);
    }

    function _ecMul(uint256 d, uint256 x1, uint256 y1, uint256 z1)
        public
        pure
        returns (uint256 x3, uint256 y3, uint256 z3)
    {
        uint256 remaining = d;
        uint256 px = x1;
        uint256 py = y1;
        uint256 pz = z1;
        uint256 acx = 0;
        uint256 acy = 0;
        uint256 acz = 1;

        if (d == 0) {
            return (0, 0, 1);
        }

        while (remaining != 0) {
            if ((remaining & 1) != 0) {
                (acx, acy, acz) = _ecAdd(acx, acy, acz, px, py, pz);
            }
            remaining = remaining / 2;
            (px, py, pz) = _ecDouble(px, py, pz);
        }

        (x3, y3, z3) = (acx, acy, acz);
    }

    function ecadd(uint256 x1, uint256 y1, uint256 x2, uint256 y2) public pure returns (uint256 x3, uint256 y3) {
        uint256 z;
        (x3, y3, z) = _ecAdd(x1, y1, 1, x2, y2, 1);
        z = _inverse(z);
        x3 = mulmod(x3, z, n);
        y3 = mulmod(y3, z, n);
    }

    function ecmul(uint256 x1, uint256 y1, uint256 scalar) public pure returns (uint256 x2, uint256 y2) {
        uint256 z;
        (x2, y2, z) = _ecMul(scalar, x1, y1, 1);
        z = _inverse(z);
        x2 = mulmod(x2, z, n);
        y2 = mulmod(y2, z, n);
    }

    function publicKey(uint256 privKey) public pure returns (uint256 qx, uint256 qy) {
        return ecmul(gx, gy, privKey);
    }

    function deriveKey(uint256 privKey, uint256 pubX, uint256 pubY) public pure returns (uint256 qx, uint256 qy) {
        uint256 z;
        (qx, qy, z) = _ecMul(privKey, pubX, pubY, 1);
        z = _inverse(z);
        qx = mulmod(qx, z, n);
        qy = mulmod(qy, z, n);
    }

    function compressPoint(uint256 x, uint256 y) public pure returns (bytes memory) {
        bytes memory compressed = new bytes(33);
        bytes1 prefix = bytes1(y % 2 == 0 ? 0x02 : 0x03);
        compressed[0] = prefix;
        assembly {
            mstore(add(compressed, 0x21), x)
        }
        return compressed;
    }

    function bytesToUint(bytes memory bin) private pure returns (uint256) {
        uint256 number;
        for (uint256 i = 0; i < bin.length; i++) {
            number = number + uint256(uint8(bin[i])) * (2 ** (8 * (bin.length - (i + 1))));
        }
        return number;
    }

    function recoverY(bytes calldata compressed) public pure returns (uint256) {
        uint8 prefix = uint8(compressed[0]);
        require(prefix == 2 || prefix == 3, "Invalid prefix");

        uint256 p = n;
        uint256 x = bytesToUint(compressed[1:33]);
        // x^3 + ax + b
        uint256 y2 = addmod(mulmod(x, mulmod(x, x, p), p), addmod(mulmod(x, a, p), b, p), p);
        uint256 y = modExp(y2, (p + 1) / 4, p);
        if ((y % 2) != prefix % 2) {
            y = p - y;
        }

        return y;
    }

    function modExp(uint256 base, uint256 exponent, uint256 modulus) public pure returns (uint256) {
        if (modulus == 1) return 0;
        uint256 result = 1;
        base = base % modulus;
        while (exponent > 0) {
            if (exponent % 2 == 1) {
                result = mulmod(result, base, modulus);
            }
            exponent = exponent >> 1;
            base = mulmod(base, base, modulus);
        }
        return result;
    }
}
