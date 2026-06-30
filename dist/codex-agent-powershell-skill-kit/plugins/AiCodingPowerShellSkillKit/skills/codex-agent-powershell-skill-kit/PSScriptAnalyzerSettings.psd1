@{
    Severity = @('Error', 'Warning')
    IncludeDefaultRules = $true
    Rules = @{
        PSUseCompatibleSyntax = @{
            Enable = $true
            TargetVersions = @('7.0', '7.1', '7.2')
        }
        PSAvoidUsingCmdletAliases = @{
            Enable = $true
        }
        PSAvoidUsingWriteHost = @{
            Enable = $true
        }
        PSUseShouldProcessForStateChangingFunctions = @{
            Enable = $true
        }
    }
}
