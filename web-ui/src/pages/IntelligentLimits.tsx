import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Chip,
  Button,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Alert,
  LinearProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  ExpandMore,
  Warning,
  CheckCircle,
  Error,
  Info,
  TrendingUp,
  TrendingDown,
  Settings,
  Refresh,
  PriorityHigh,
  PriorityLow,
} from '@mui/icons-material';
import { useIntelligentLimits } from '../api/useIntelligentLimits';

interface IntelligentLimitRecommendation {
  limit_name: string;
  category: string;
  current_value: any;
  recommended_value: any;
  observed_peak: number;
  average_usage: number;
  usage_percentile_95: number;
  usage_percentile_99: number;
  risk_level: string;
  confidence: number;
  reason: string;
  impact: string;
  priority: string;
  estimated_savings: any;
  implementation_steps: string[];
  last_updated: string;
}

interface TenantAnalysis {
  tenant_name: string;
  risk_score: number;
  reliability_score: number;
  performance_score: number;
  cost_optimization_score: number;
  recommendations: IntelligentLimitRecommendation[];
  missing_limits: string[];
  summary: any;
}

const IntelligentLimits: React.FC = () => {
  const [selectedTenant, setSelectedTenant] = useState<string>('');
  const [updateDialogOpen, setUpdateDialogOpen] = useState(false);
  const [selectedLimit, setSelectedLimit] = useState<IntelligentLimitRecommendation | null>(null);
  const [newValue, setNewValue] = useState<string>('');
  const [updateReason, setUpdateReason] = useState<string>('');

  const {
    data: recommendationsData,
    loading,
    error,
    refetch,
  } = useIntelligentLimits();

  const getPriorityIcon = (priority: string) => {
    switch (priority) {
      case 'critical':
        return <PriorityHigh color="error" />;
      case 'high':
        return <PriorityHigh color="warning" />;
      case 'medium':
        return <Info />;
      case 'low':
        return <PriorityLow color="success" />;
      default:
        return <Info />;
    }
  };

  const getRiskLevelColor = (riskLevel: string) => {
    switch (riskLevel) {
      case 'critical':
        return 'error';
      case 'high':
        return 'warning';
      case 'medium':
        return 'info';
      case 'low':
        return 'success';
      default:
        return 'default';
    }
  };

  const getScoreColor = (score: number) => {
    if (score >= 0.8) return 'success';
    if (score >= 0.6) return 'warning';
    return 'error';
  };

  const handleUpdateLimit = (recommendation: IntelligentLimitRecommendation) => {
    setSelectedLimit(recommendation);
    setNewValue(String(recommendation.recommended_value));
    setUpdateDialogOpen(true);
  };

  const handleUpdateSubmit = async () => {
    if (!selectedLimit || !selectedTenant) return;

    try {
      const response = await fetch(`/api/tenants/${selectedTenant}/limits`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          limit_name: selectedLimit.limit_name,
          new_value: newValue,
          reason: updateReason,
          apply_now: true,
        }),
      });

      if (response.ok) {
        setUpdateDialogOpen(false);
        refetch();
      } else {
        console.error('Failed to update limit');
      }
    } catch (error) {
      console.error('Error updating limit:', error);
    }
  };

  if (loading) {
    return (
      <Box sx={{ width: '100%' }}>
        <LinearProgress />
        <Typography variant="h6" sx={{ mt: 2 }}>
          Analyzing tenant limits...
        </Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        Failed to load intelligent limit recommendations: {error.message}
      </Alert>
    );
  }

  const tenantRecommendations = recommendationsData?.tenant_recommendations || [];
  const overallSummary = recommendationsData?.overall_summary || {};
  const averageScores = recommendationsData?.average_scores || {};

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1">
          Intelligent Limit Analysis
        </Typography>
        <Button
          variant="contained"
          startIcon={<Refresh />}
          onClick={() => refetch()}
        >
          Refresh Analysis
        </Button>
      </Box>

      {/* Overall Summary */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Overall Summary
          </Typography>
          <Grid container spacing={3}>
            <Grid item xs={12} md={3}>
              <Box textAlign="center">
                <Typography variant="h4" color={getScoreColor(averageScores.risk_score)}>
                  {Math.round(averageScores.risk_score * 100)}%
                </Typography>
                <Typography variant="body2" color="textSecondary">
                  Average Risk Score
                </Typography>
              </Box>
            </Grid>
            <Grid item xs={12} md={3}>
              <Box textAlign="center">
                <Typography variant="h4" color={getScoreColor(averageScores.reliability_score)}>
                  {Math.round(averageScores.reliability_score * 100)}%
                </Typography>
                <Typography variant="body2" color="textSecondary">
                  Average Reliability Score
                </Typography>
              </Box>
            </Grid>
            <Grid item xs={12} md={3}>
              <Box textAlign="center">
                <Typography variant="h4" color={getScoreColor(averageScores.performance_score)}>
                  {Math.round(averageScores.performance_score * 100)}%
                </Typography>
                <Typography variant="body2" color="textSecondary">
                  Average Performance Score
                </Typography>
              </Box>
            </Grid>
            <Grid item xs={12} md={3}>
              <Box textAlign="center">
                <Typography variant="h4" color={getScoreColor(averageScores.cost_optimization_score)}>
                  {Math.round(averageScores.cost_optimization_score * 100)}%
                </Typography>
                <Typography variant="body2" color="textSecondary">
                  Cost Optimization Score
                </Typography>
              </Box>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Recommendations Summary */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            Recommendations Summary
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={6} md={3}>
              <Chip
                icon={<PriorityHigh />}
                label={`${overallSummary.critical_recommendations || 0} Critical`}
                color="error"
                variant="outlined"
              />
            </Grid>
            <Grid item xs={6} md={3}>
              <Chip
                icon={<PriorityHigh />}
                label={`${overallSummary.high_priority_recommendations || 0} High Priority`}
                color="warning"
                variant="outlined"
              />
            </Grid>
            <Grid item xs={6} md={3}>
              <Chip
                icon={<Warning />}
                label={`${overallSummary.missing_limits_total || 0} Missing Limits`}
                color="error"
                variant="outlined"
              />
            </Grid>
            <Grid item xs={6} md={3}>
              <Chip
                icon={<TrendingUp />}
                label={`${overallSummary.cost_optimization_opportunities || 0} Cost Opportunities`}
                color="success"
                variant="outlined"
              />
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* Tenant Analysis */}
      {tenantRecommendations.map((tenantAnalysis: TenantAnalysis) => (
        <Card key={tenantAnalysis.tenant_name} sx={{ mb: 2 }}>
          <CardContent>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="h6">
                {tenantAnalysis.tenant_name}
              </Typography>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Chip
                  label={`Risk: ${Math.round(tenantAnalysis.risk_score * 100)}%`}
                  color={getRiskLevelColor(tenantAnalysis.risk_score > 0.7 ? 'high' : 'low')}
                  size="small"
                />
                <Chip
                  label={`Reliability: ${Math.round(tenantAnalysis.reliability_score * 100)}%`}
                  color={getScoreColor(tenantAnalysis.reliability_score)}
                  size="small"
                />
              </Box>
            </Box>

            {tenantAnalysis.recommendations.length > 0 && (
              <Accordion>
                <AccordionSummary expandIcon={<ExpandMore />}>
                  <Typography>
                    {tenantAnalysis.recommendations.length} Recommendations
                  </Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <List>
                    {tenantAnalysis.recommendations.map((recommendation, index) => (
                      <React.Fragment key={index}>
                        <ListItem>
                          <ListItemIcon>
                            {getPriorityIcon(recommendation.priority)}
                          </ListItemIcon>
                          <ListItemText
                            primary={
                              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                <Typography variant="subtitle1">
                                  {recommendation.limit_name}
                                </Typography>
                                <Box sx={{ display: 'flex', gap: 1 }}>
                                  <Chip
                                    label={recommendation.priority}
                                    color={getRiskLevelColor(recommendation.risk_level)}
                                    size="small"
                                  />
                                  <Chip
                                    label={recommendation.category}
                                    variant="outlined"
                                    size="small"
                                  />
                                </Box>
                              </Box>
                            }
                            secondary={
                              <Box>
                                <Typography variant="body2" color="textSecondary">
                                  {recommendation.reason}
                                </Typography>
                                <Box sx={{ mt: 1, display: 'flex', gap: 2 }}>
                                  <Typography variant="body2">
                                    Current: {recommendation.current_value || 'Not set'}
                                  </Typography>
                                  <Typography variant="body2">
                                    Recommended: {recommendation.recommended_value}
                                  </Typography>
                                  <Typography variant="body2">
                                    Peak: {recommendation.observed_peak.toFixed(2)}
                                  </Typography>
                                </Box>
                                <Box sx={{ mt: 1 }}>
                                  <Typography variant="body2" color="textSecondary">
                                    Implementation Steps:
                                  </Typography>
                                  <List dense>
                                    {recommendation.implementation_steps.map((step, stepIndex) => (
                                      <ListItem key={stepIndex} sx={{ py: 0 }}>
                                        <ListItemText
                                          primary={step}
                                          primaryTypographyProps={{ variant: 'body2' }}
                                        />
                                      </ListItem>
                                    ))}
                                  </List>
                                </Box>
                              </Box>
                            }
                          />
                          <Box sx={{ display: 'flex', gap: 1 }}>
                            <Tooltip title="Update Limit">
                              <IconButton
                                size="small"
                                onClick={() => {
                                  setSelectedTenant(tenantAnalysis.tenant_name);
                                  handleUpdateLimit(recommendation);
                                }}
                              >
                                <Settings />
                              </IconButton>
                            </Tooltip>
                          </Box>
                        </ListItem>
                        {index < tenantAnalysis.recommendations.length - 1 && <Divider />}
                      </React.Fragment>
                    ))}
                  </List>
                </AccordionDetails>
              </Accordion>
            )}

            {tenantAnalysis.missing_limits.length > 0 && (
              <Alert severity="warning" sx={{ mt: 2 }}>
                <Typography variant="subtitle2" gutterBottom>
                  Missing Limits ({tenantAnalysis.missing_limits.length}):
                </Typography>
                <Typography variant="body2">
                  {tenantAnalysis.missing_limits.join(', ')}
                </Typography>
              </Alert>
            )}
          </CardContent>
        </Card>
      ))}

      {/* Update Limit Dialog */}
      <Dialog open={updateDialogOpen} onClose={() => setUpdateDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>
          Update Limit: {selectedLimit?.limit_name}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ mt: 2 }}>
            <TextField
              fullWidth
              label="New Value"
              value={newValue}
              onChange={(e) => setNewValue(e.target.value)}
              sx={{ mb: 2 }}
            />
            <TextField
              fullWidth
              label="Reason for Update"
              value={updateReason}
              onChange={(e) => setUpdateReason(e.target.value)}
              multiline
              rows={3}
            />
            {selectedLimit && (
              <Alert severity="info" sx={{ mt: 2 }}>
                <Typography variant="body2">
                  Current: {selectedLimit.current_value || 'Not set'} → Recommended: {selectedLimit.recommended_value}
                </Typography>
              </Alert>
            )}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setUpdateDialogOpen(false)}>
            Cancel
          </Button>
          <Button onClick={handleUpdateSubmit} variant="contained">
            Update Limit
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default IntelligentLimits; 