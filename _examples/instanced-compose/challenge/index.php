<!DOCTYPE html>
<html>
<head>
    <title>Secret Vault - Login</title>
    <style>
        body {
            font-family: 'Segoe UI', Arial, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            margin: 0;
            color: #fff;
        }
        .container {
            background: rgba(255, 255, 255, 0.1);
            padding: 40px;
            border-radius: 15px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
            backdrop-filter: blur(10px);
            width: 350px;
        }
        h1 {
            text-align: center;
            margin-bottom: 30px;
            color: #00d9ff;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 8px;
            color: #aaa;
        }
        input[type="text"], input[type="password"] {
            width: 100%;
            padding: 12px;
            border: none;
            border-radius: 8px;
            background: rgba(255, 255, 255, 0.1);
            color: #fff;
            font-size: 16px;
            box-sizing: border-box;
        }
        input[type="submit"] {
            width: 100%;
            padding: 14px;
            border: none;
            border-radius: 8px;
            background: #00d9ff;
            color: #1a1a2e;
            font-size: 16px;
            font-weight: bold;
            cursor: pointer;
            transition: background 0.3s;
        }
        input[type="submit"]:hover {
            background: #00b8d4;
        }
        .error {
            background: rgba(255, 0, 0, 0.2);
            padding: 10px;
            border-radius: 8px;
            margin-bottom: 20px;
            text-align: center;
        }
        .success {
            background: rgba(0, 255, 0, 0.2);
            padding: 10px;
            border-radius: 8px;
            margin-bottom: 20px;
            text-align: center;
        }
        .hint {
            text-align: center;
            margin-top: 20px;
            color: #666;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Secret Vault</h1>
        
        <?php
        $error = '';
        $success = '';
        
        if ($_SERVER['REQUEST_METHOD'] === 'POST') {
            $host = getenv('DB_HOST') ?: 'db';
            $user = getenv('DB_USER') ?: 'challenge';
            $pass = getenv('DB_PASS') ?: 'challengepass';
            $dbname = getenv('DB_NAME') ?: 'ctf';
            
            $conn = new mysqli($host, $user, $pass, $dbname);
            
            if ($conn->connect_error) {
                $error = "Connection failed. Please try again.";
            } else {
                $username = $_POST['username'];
                $password = $_POST['password'];
                
                // VULNERABLE: SQL Injection!
                $query = "SELECT * FROM users WHERE username='$username' AND password='$password'";
                $result = $conn->query($query);
                
                if ($result && $result->num_rows > 0) {
                    $row = $result->fetch_assoc();
                    
                    if ($row['role'] === 'admin') {
                        // Admin login - show secrets
                        $secrets_query = "SELECT * FROM secrets";
                        $secrets_result = $conn->query($secrets_query);
                        
                        $success = "Welcome Admin! Here are your secrets:<br><br>";
                        while ($secret = $secrets_result->fetch_assoc()) {
                            $success .= "<b>" . htmlspecialchars($secret['secret_name']) . ":</b> " . 
                                        htmlspecialchars($secret['secret_value']) . "<br>";
                        }
                    } else {
                        $success = "Welcome, " . htmlspecialchars($row['username']) . "! You're logged in as a regular user.";
                    }
                } else {
                    $error = "Invalid username or password!";
                }
                
                $conn->close();
            }
        }
        ?>
        
        <?php if ($error): ?>
            <div class="error"><?php echo $error; ?></div>
        <?php endif; ?>
        
        <?php if ($success): ?>
            <div class="success"><?php echo $success; ?></div>
        <?php else: ?>
            <form method="POST">
                <div class="form-group">
                    <label>Username</label>
                    <input type="text" name="username" required placeholder="Enter username">
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" name="password" required placeholder="Enter password">
                </div>
                <input type="submit" value="Login">
            </form>
        <?php endif; ?>
        
        <p class="hint">Hint: Try logging in as admin to see the secrets!</p>
    </div>
</body>
</html>
